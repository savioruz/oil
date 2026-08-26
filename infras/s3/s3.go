// Package s3 provides S3-compatible client utilities for file storage operations.
//
//nolint:revive
package s3

//go:generate go run go.uber.org/mock/mockgen -source=./s3.go -destination=./mocks/s3_mock.go -package=mocks

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"

	"github.com/savioruz/oil/config"
	"github.com/savioruz/oil/infras/otel"
	"github.com/savioruz/oil/shared/constant"
	"mime/multipart"
)

const (
	otelAttrFileName = "file_name"
	otelAttrBucket   = "bucket"
)

// S3 defines the interface for S3 operations.
type S3 interface {
	UploadFile(ctx context.Context, bucketName, directory string, file multipart.File, fileHeader *multipart.FileHeader, fileName string) (url string, err error)
	UploadFileBytes(ctx context.Context, bucketName, directory, fileName, contentType string, fileData []byte) (url string, err error)
	DeleteFile(ctx context.Context, bucketName, directory, objectName string) error
	GetObjectNameFromURL(bucketName, url string) (objectName string)
	GetPresignedUploadURL(ctx context.Context, bucketName, directory, fileName, contentType string, expiryMinutes int) (string, error)
}

type s3Impl struct {
	Client *minio.Client
	Config *config.Config
	otel   otel.Otel
}

func (svc *s3Impl) UploadFile(ctx context.Context, bucketName, directory string, file multipart.File, fileHeader *multipart.FileHeader, fileName string) (url string, err error) {
	ctx, scope := svc.otel.NewScope(ctx, constant.OtelS3ScopeName, constant.OtelS3ScopeName+".UploadFile")
	defer scope.End()
	defer scope.TraceIfError(err)

	if bucketName == "" {
		bucketName = svc.Config.External.S3.BucketName
	}

	scope.SetAttribute(otelAttrFileName, fileName)
	scope.SetAttribute(otelAttrBucket, bucketName)

	buf := bytes.NewBuffer(nil)

	if _, err = buf.ReadFrom(file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	contentType := fileHeader.Header.Get(constant.RequestHeaderContentType)

	return svc.upload(ctx, bucketName, directory, fileName, contentType, buf)
}

func (svc *s3Impl) UploadFileBytes(ctx context.Context, bucketName, directory, fileName, contentType string, fileData []byte) (url string, err error) {
	ctx, scope := svc.otel.NewScope(ctx, constant.OtelS3ScopeName, constant.OtelS3ScopeName+".UploadFileBytes")
	defer scope.End()
	defer scope.TraceIfError(err)

	if bucketName == "" {
		bucketName = svc.Config.External.S3.BucketName
	}

	scope.SetAttribute(otelAttrFileName, fileName)
	scope.SetAttribute(otelAttrBucket, bucketName)

	buf := bytes.NewBuffer(fileData)

	return svc.upload(ctx, bucketName, directory, fileName, contentType, buf)
}

func (svc *s3Impl) DeleteFile(ctx context.Context, bucketName, directory, objectName string) (err error) {
	ctx, scope := svc.otel.NewScope(ctx, constant.OtelS3ScopeName, constant.OtelS3ScopeName+".DeleteFile")
	defer scope.End()
	defer scope.TraceIfError(err)

	scope.SetAttribute(otelAttrFileName, objectName)
	scope.SetAttribute(otelAttrBucket, bucketName)

	objectKey := path.Join(directory, objectName)

	err = svc.Client.RemoveObject(ctx, bucketName, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		log.Error().Err(err).Msg("failed to delete file from S3")

		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

func (svc *s3Impl) GetObjectNameFromURL(bucketName, url string) (objectName string) {
	publicDomain := svc.Config.External.S3.PublicDomain

	bucketPrefix := path.Join(publicDomain, bucketName) + "/"
	if len(url) >= len(bucketPrefix) && url[:len(bucketPrefix)] == bucketPrefix {
		return url[len(bucketPrefix):]
	}

	apiEndpoint := svc.Config.External.S3.APIEndpoint

	bucketURL := apiEndpoint + "/" + bucketName + "/"
	if len(url) >= len(bucketURL) && url[:len(bucketURL)] == bucketURL {
		return url[len(bucketURL):]
	}

	return ""
}

func (svc *s3Impl) GetPresignedUploadURL(ctx context.Context, bucketName, directory, fileName, contentType string, expiryMinutes int) (string, error) {
	ctx, scope := svc.otel.NewScope(ctx, constant.OtelS3ScopeName, constant.OtelS3ScopeName+".GetPresignedUploadURL")
	defer scope.End()

	if bucketName == "" {
		bucketName = svc.Config.External.S3.BucketName
	}

	objectKey := path.Join(directory, fileName)

	// NOTE: minio-go's PresignedPutObject does not carry ContentType through
	// the signature the way s3.PutObjectInput did. If you need to constrain
	// the uploader to a specific content-type, enforce it server-side after
	// upload (HEAD the object) or use PresignedPostPolicy with a content-type
	// condition instead of PresignedPutObject.
	_ = contentType

	presignedURL, err := svc.Client.PresignedPutObject(ctx, bucketName, objectKey, time.Duration(expiryMinutes)*time.Minute)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate presigned URL")

		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL.String(), nil
}

func (svc *s3Impl) upload(ctx context.Context, bucket, directory, fileName, contentType string, buf *bytes.Buffer) (url string, err error) {
	ctx, scope := svc.otel.NewScope(ctx, constant.OtelS3ScopeName, constant.OtelS3ScopeName+".upload")
	defer scope.End()
	defer scope.TraceIfError(err)

	objectKey := path.Join(directory, fileName)
	size := int64(buf.Len())

	_, err = svc.Client.PutObject(ctx, bucket, objectKey, buf, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	publicDomain := svc.Config.External.S3.PublicDomain

	return publicDomain + "/" + objectKey, nil
}

func New(cfg *config.Config, otl otel.Otel) S3 {
	rawEndpoint := cfg.External.S3.APIEndpoint
	accessKeyID := cfg.External.S3.AccessKeyID
	secretAccessKey := cfg.External.S3.SecretAccessKey

	secure := true
	endpoint := rawEndpoint

	if after, ok := strings.CutPrefix(rawEndpoint, "https://"); ok {
		endpoint = after
	} else if after, ok := strings.CutPrefix(rawEndpoint, "http://"); ok {
		endpoint = after
		secure = false
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: secure,
	})
	if err != nil {
		log.Err(err).Msg("Error creating minio client")
	}

	return &s3Impl{
		Client: client,
		Config: cfg,
		otel:   otl,
	}
}
