package service

import (
	"context"
	"fmt"
	"github.com/savioruz/oil/config"
	"github.com/savioruz/oil/infras/otel"
	"github.com/savioruz/oil/infras/s3"
	"github.com/savioruz/oil/internal/modules/userprofile/model"
	"github.com/savioruz/oil/internal/modules/userprofile/model/dto"
	"github.com/savioruz/oil/internal/modules/userprofile/repository"
	"github.com/savioruz/oil/shared"
	"github.com/savioruz/oil/shared/constant"
	"github.com/savioruz/oil/shared/errkey"
	"github.com/savioruz/oil/shared/failure"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	presignedURLExpiryMinutes = 5
)

type Userprofile interface {
	GetOrCreateByAuthUserID(ctx context.Context, authUserID, email, role string) (dto.UserprofileResponse, error)
	Get(ctx context.Context, id string) (dto.UserprofileResponse, error)
	Update(ctx context.Context, req dto.UpdateUserprofileRequest, id string) error
	GeneratePresignedURL(ctx context.Context, req dto.GeneratePresignedURLRequest, userID string) (dto.GeneratePresignedURLResponse, error)
}

type serviceImpl struct {
	repo repository.Userprofile
	cfg  *config.Config
	s3   s3.S3
	otel otel.Otel
}

func New(repo repository.Userprofile, cfg *config.Config, s3 s3.S3, otel otel.Otel) Userprofile {
	return &serviceImpl{
		repo: repo,
		cfg:  cfg,
		s3:   s3,
		otel: otel,
	}
}

func (s *serviceImpl) GetOrCreateByAuthUserID(ctx context.Context, authUserID, email, role string) (res dto.UserprofileResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".GetOrCreateByAuthUserID")
	defer scope.End()
	defer scope.TraceIfError(err)

	filter := shared.SingleFilter(authUserID, model.FieldAuthUserID, model.TableName)

	profile, err := s.repo.Get(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to get userprofile")

		return res, failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to get userprofile: %v", err))
	}

	if profile.ID != "" {
		res.FromModel(profile)

		return res, nil
	}

	req := dto.CreateUserprofileRequest{
		AuthUserID: authUserID,
		Email:      email,
		Role:       role,
	}

	err = s.repo.Insert(ctx, req.ToModel())
	if err != nil {
		log.Error().Err(err).Msg("failed to create userprofile")

		return res, failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to create userprofile: %v", err))
	}

	profile, err = s.repo.Get(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to get userprofile after creation")

		return res, failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to get userprofile: %v", err))
	}

	res.FromModel(profile)

	return res, nil
}

func (s *serviceImpl) Get(ctx context.Context, id string) (res dto.UserprofileResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Get")
	defer scope.End()
	defer scope.TraceIfError(err)

	filter := shared.SingleFilter(id, model.FieldID, model.TableName)

	profile, err := s.repo.Get(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to get userprofile")

		return res, failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to get userprofile: %v", err))
	}

	if profile.ID == "" {
		return res, failure.NotFoundWithKey(errkey.ErrUserprofileNotFound, "userprofile not found")
	}

	res.FromModel(profile)

	return res, nil
}

func (s *serviceImpl) Update(ctx context.Context, req dto.UpdateUserprofileRequest, id string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Update")
	defer scope.End()
	defer scope.TraceIfError(err)

	filter := shared.SingleFilter(id, model.FieldID, model.TableName)

	exist, err := s.repo.Exist(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if userprofile exists")

		return failure.InternalErrorWithKey(errkey.ErrDatabaseQuery, fmt.Sprintf("failed to check if userprofile exists: %v", err))
	}

	if !exist {
		log.Error().Msg("userprofile not found")

		return failure.NotFoundWithKey(errkey.ErrUserprofileNotFound, "userprofile not found")
	}

	updatedFields := shared.TransformFields(req, id)
	if len(updatedFields) == 0 {
		return nil
	}

	err = s.repo.Update(ctx, updatedFields, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to update userprofile")

		return failure.InternalErrorWithKey(errkey.ErrUserprofileUpdateFailed, fmt.Sprintf("failed to update userprofile: %v", err))
	}

	return nil
}

func (s *serviceImpl) GeneratePresignedURL(ctx context.Context, req dto.GeneratePresignedURLRequest, userID string) (res dto.GeneratePresignedURLResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".GeneratePresignedURL")
	defer scope.End()
	defer scope.TraceIfError(err)

	fileKey := fmt.Sprintf("userprofiles/%s/%d-%s", userID, time.Now().Unix(), req.FileName)

	uploadURL, err := s.s3.GetPresignedUploadURL(
		ctx,
		s.cfg.External.S3.BucketName,
		model.EntityName,
		fileKey,
		req.ContentType,
		presignedURLExpiryMinutes,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate presigned URL")

		return dto.GeneratePresignedURLResponse{}, failure.InternalError(err)
	}

	return dto.GeneratePresignedURLResponse{
		UploadURL: uploadURL,
		FileKey:   fileKey,
	}, nil
}
