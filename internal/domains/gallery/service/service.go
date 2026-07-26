package service

import (
	"context"
	"errors"
	"fmt"
	"oil/config"
	"oil/infras/otel"
	"oil/infras/s3"
	"oil/internal/domains/gallery/model"
	"oil/internal/domains/gallery/model/dto"
	"oil/internal/domains/gallery/repository"
	"oil/shared"
	"oil/shared/cache"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/errkey"
	"oil/shared/failure"
	"oil/shared/singleflight"

	"github.com/rs/zerolog/log"
)

const (
	cacheGetGallery    = "gallery:get"
	cacheGetAllGallery = "gallery:get_all"
	cacheCountGallery  = "gallery:count"
)

var (
	ErrDeleteImagesFromS3 = errors.New("failed to delete images from S3")
)

type Gallery interface {
	Create(ctx context.Context, req dto.CreateGalleryRequest) error
	GetAll(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (dto.GetGalleriesResponse, error)
	Count(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (int, error)
	Get(ctx context.Context, id string) (dto.GalleryResponse, error)
	Update(ctx context.Context, req dto.UpdateGalleryRequest, id string) error
	Delete(ctx context.Context, id string) error
	UploadImage(ctx context.Context, req dto.UploadImageRequest) (dto.UploadImageResponse, error)
	DeleteImagesFromS3(ctx context.Context, req dto.DeleteImagesRequest) error
}

type serviceImpl struct {
	repo  repository.Gallery
	cfg   *config.Config
	cache cache.RedisCache
	otel  otel.Otel
	s3    s3.S3
	sf    *singleflight.Group
}

func New(repo repository.Gallery, cfg *config.Config, cache cache.RedisCache, otel otel.Otel, s3 s3.S3, sf *singleflight.Group) Gallery {
	return &serviceImpl{
		repo:  repo,
		cfg:   cfg,
		cache: cache,
		otel:  otel,
		s3:    s3,
		sf:    sf,
	}
}

func (s *serviceImpl) Create(ctx context.Context, req dto.CreateGalleryRequest) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Create")
	defer scope.End()
	defer scope.TraceIfError(err)

	user := shared.GetUserID(ctx)

	if err = s.repo.Insert(ctx, req.ToModel(user)); err != nil {
		return failure.InternalErrorWithKey(errkey.ErrGalleryCreateFailed, fmt.Sprintf("failed to create gallery: %v", err))
	}

	go func() {
		c := context.WithoutCancel(ctx)

		shared.InvalidateCaches(c, s.cache, cacheGetAllGallery)
		shared.InvalidateCaches(c, s.cache, cacheCountGallery)
	}()

	return nil
}

func (s *serviceImpl) GetAll(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (res dto.GetGalleriesResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".GetAll")
	defer scope.End()
	defer scope.TraceIfError(err)

	cacheKey := shared.BuildCacheKeyWithQuery(cacheGetAllGallery, req, filter)

	return cache.Remember(ctx, s.cache, s.sf, cacheKey, s.cfg.Cache.TTL, func(ctx context.Context) (dto.GetGalleriesResponse, error) {
		var out dto.GetGalleriesResponse

		total, err := s.Count(ctx, req, filter)
		if err != nil {
			log.Error().Err(err).Msg("failed to count galleries")

			return out, failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to count galleries: %v", err))
		}

		galleries, err := s.repo.GetAll(ctx, req, filter)
		if err != nil {
			log.Error().Err(err).Msg("failed to get galleries")

			return out, failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to get galleries: %v", err))
		}

		out.FromModels(galleries, total, req.Limit)

		return out, nil
	})
}

func (s *serviceImpl) Count(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (total int, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Count")
	defer scope.End()
	defer scope.TraceIfError(err)

	cacheKey := shared.BuildCacheKeyWithQuery(cacheCountGallery, req, filter)

	return cache.Remember(ctx, s.cache, s.sf, cacheKey, s.cfg.Cache.TTL, func(ctx context.Context) (int, error) {
		count, err := s.repo.Count(ctx, filter)
		if err != nil {
			log.Error().Err(err).Msg("failed to count galleries")

			return 0, failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to count galleries: %v", err))
		}

		return count, nil
	})
}

func (s *serviceImpl) Get(ctx context.Context, id string) (res dto.GalleryResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Get")
	defer scope.End()
	defer scope.TraceIfError(err)

	cacheKey := shared.BuildCacheKey(cacheGetGallery, id)

	return cache.Remember(ctx, s.cache, s.sf, cacheKey, s.cfg.Cache.TTL, func(ctx context.Context) (dto.GalleryResponse, error) {
		var out dto.GalleryResponse

		gallery, err := s.repo.Get(ctx, shared.SingleFilter(id, model.FieldID, model.TableName))
		if err != nil {
			log.Error().Err(err).Msg("failed to get gallery")

			return out, failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to get gallery: %v", err))
		}

		if gallery.ID == "" {
			return out, failure.NotFoundWithKey(errkey.ErrGalleryNotFound, "gallery not found")
		}

		out.FromModel(gallery)

		return out, nil
	})
}

func (s *serviceImpl) Update(ctx context.Context, req dto.UpdateGalleryRequest, id string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Update")
	defer scope.End()
	defer scope.TraceIfError(err)

	user := shared.GetUserID(ctx)
	filter := shared.SingleFilter(id, model.FieldID, model.TableName)

	exist, err := s.repo.Exist(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to check gallery existence")

		return failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to check gallery existence: %v", err))
	}

	if !exist {
		log.Error().Msg("gallery not found")

		return failure.NotFoundWithKey(errkey.ErrGalleryNotFound, "gallery not found")
	}

	updatedFields := shared.TransformFields(req, user)
	if err = s.repo.Update(ctx, updatedFields, filter); err != nil {
		log.Error().Err(err).Msg("failed to update gallery")

		return failure.InternalErrorWithKey(errkey.ErrGalleryUpdateFailed, fmt.Sprintf("failed to update gallery: %v", err))
	}

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Delete(c, shared.BuildCacheKey(cacheGetGallery, id)); err != nil {
			log.Error().Err(err).Msg("failed to delete gallery cache")
		}

		shared.InvalidateCaches(c, s.cache, cacheGetAllGallery)
		shared.InvalidateCaches(c, s.cache, cacheCountGallery)
	}()

	return nil
}

func (s *serviceImpl) Delete(ctx context.Context, id string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Delete")
	defer scope.End()
	defer scope.TraceIfError(err)

	filter := shared.SingleFilter(id, model.FieldID, model.TableName)

	gallery, err := s.repo.Get(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to get gallery for image deletion")

		return failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to get gallery: %v", err))
	}

	if gallery.ID == "" {
		log.Error().Msg("gallery not found")

		return failure.NotFoundWithKey(errkey.ErrGalleryNotFound, "gallery not found")
	}

	if err = s.repo.Delete(ctx, filter); err != nil {
		log.Error().Err(err).Msg("failed to delete gallery")

		return failure.InternalErrorWithKey(errkey.ErrGalleryDeleteFailed, fmt.Sprintf("failed to delete gallery: %v", err))
	}

	images := make([]string, len(gallery.Images))
	copy(images, gallery.Images)

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Delete(c, shared.BuildCacheKey(cacheGetGallery, id)); err != nil {
			log.Error().Err(err).Msg("failed to delete gallery cache")
		}

		shared.InvalidateCaches(c, s.cache, cacheGetAllGallery)
		shared.InvalidateCaches(c, s.cache, cacheCountGallery)

		if len(images) > 0 {
			deleteReq := dto.DeleteImagesRequest{
				ImageURLs: images,
			}
			if err := s.DeleteImagesFromS3(c, deleteReq); err != nil {
				log.Error().Err(err).Msg("failed to delete images from S3")
			}
		}
	}()

	return nil
}

func (s *serviceImpl) UploadImage(ctx context.Context, req dto.UploadImageRequest) (res dto.UploadImageResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".UploadImage")
	defer scope.End()
	defer scope.TraceIfError(err)

	bucketName := s.cfg.External.S3.BucketName

	url, err := s.s3.UploadFile(ctx, bucketName, model.EntityName, req.ImageFile, req.Image, req.Image.Filename)
	if err != nil {
		log.Error().Err(err).Msg("failed to upload file to S3")

		return res, failure.InternalErrorWithKey(errkey.ErrS3Upload, fmt.Sprintf("failed to upload file to S3: %v", err))
	}

	res.FromModel(url, req.Image.Filename)

	return res, nil
}

func (s *serviceImpl) DeleteImagesFromS3(ctx context.Context, req dto.DeleteImagesRequest) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".DeleteImagesFromS3")
	defer scope.End()
	defer scope.TraceIfError(err)

	bucketName := s.cfg.External.S3.BucketName

	var deleteErrors []error

	for _, imageURL := range req.ImageURLs {
		objectName := s.s3.GetObjectNameFromURL(bucketName, imageURL)
		if objectName == "" {
			log.Warn().Str("url", imageURL).Msg("failed to extract object name from URL")

			continue
		}

		if err := s.s3.DeleteFile(ctx, bucketName, model.EntityName, objectName); err != nil {
			log.Error().Err(err).Str("objectName", objectName).Msg("failed to delete file from S3")
			deleteErrors = append(deleteErrors, err)
		}
	}

	if len(deleteErrors) > 0 {
		return failure.InternalErrorWithKey(errkey.ErrS3Delete, fmt.Sprintf("failed to delete %d images from S3", len(deleteErrors)))
	}

	return nil
}
