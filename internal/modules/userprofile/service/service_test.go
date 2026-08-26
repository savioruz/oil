package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"oil/config"
	"oil/infras/otel"
	otelMocks "oil/infras/otel/mocks"
	s3mocks "oil/infras/s3/mocks"
	userprofileMocks "oil/internal/modules/userprofile/mocks"
	"oil/internal/modules/userprofile/model"
	"oil/internal/modules/userprofile/model/dto"
	"oil/internal/modules/userprofile/service"
	"oil/shared"
	"oil/shared/errkey"
	"oil/shared/failure"
)

func setup(t *testing.T) (*gomock.Controller, *userprofileMocks.MockUserprofile, *config.Config, *s3mocks.MockS3, otel.Otel) {
	ctrl := gomock.NewController(t)
	mockRepo := userprofileMocks.NewMockUserprofile(ctrl)
	mockOtel := otelMocks.NewOtel()
	mockS3 := s3mocks.NewMockS3(ctrl)
	cfg := &config.Config{}
	return ctrl, mockRepo, cfg, mockS3, mockOtel
}

func TestGetOrCreateByAuthUserID(t *testing.T) {
	ctrl, mockRepo, cfg, mockS3, mockOtel := setup(t)
	defer ctrl.Finish()

	svc := service.New(mockRepo, cfg, mockS3, mockOtel)

	tests := []struct {
		name       string
		authUserID string
		email      string
		role       string
		setupMock  func()
		wantErr    bool
	}{
		{
			name:       "success - profile exists",
			authUserID: "auth-123",
			email:      "test@example.com",
			role:       "user",
			setupMock: func() {
				filter := shared.SingleFilter("auth-123", model.FieldAuthUserID, model.TableName)
				mockRepo.EXPECT().Get(gomock.Any(), filter).Return(model.Userprofile{
					ID:         "profile-1",
					AuthUserID: "auth-123",
					Email:      "test@example.com",
					Role:       "user",
				}, nil)
			},
			wantErr: false,
		},
		{
			name:       "success - create new profile",
			authUserID: "auth-new",
			email:      "new@example.com",
			role:       "user",
			setupMock: func() {
				filter := shared.SingleFilter("auth-new", model.FieldAuthUserID, model.TableName)
				mockRepo.EXPECT().Get(gomock.Any(), filter).Return(model.Userprofile{}, nil)
				mockRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)
				mockRepo.EXPECT().Get(gomock.Any(), filter).Return(model.Userprofile{
					ID:         "profile-new",
					AuthUserID: "auth-new",
					Email:      "new@example.com",
					Role:       "user",
				}, nil)
			},
			wantErr: false,
		},
		{
			name:       "error - database query failed",
			authUserID: "auth-123",
			email:      "test@example.com",
			role:       "user",
			setupMock: func() {
				filter := shared.SingleFilter("auth-123", model.FieldAuthUserID, model.TableName)
				mockRepo.EXPECT().Get(gomock.Any(), filter).Return(model.Userprofile{}, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := svc.GetOrCreateByAuthUserID(context.Background(), tt.authUserID, tt.email, tt.role)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result.ID)
			}
		})
	}
}

func TestGet(t *testing.T) {
	ctrl, mockRepo, cfg, mockS3, mockOtel := setup(t)
	defer ctrl.Finish()

	svc := service.New(mockRepo, cfg, mockS3, mockOtel)

	tests := []struct {
		name      string
		id        string
		setupMock func()
		wantErr   bool
		errKey    errkey.ErrorKey
	}{
		{
			name: "success",
			id:   "profile-1",
			setupMock: func() {
				filter := shared.SingleFilter("profile-1", model.FieldID, model.TableName)
				mockRepo.EXPECT().Get(gomock.Any(), filter).Return(model.Userprofile{
					ID:         "profile-1",
					AuthUserID: "auth-123",
					Email:      "test@example.com",
					Role:       "user",
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "error - not found",
			id:   "profile-nonexistent",
			setupMock: func() {
				filter := shared.SingleFilter("profile-nonexistent", model.FieldID, model.TableName)
				mockRepo.EXPECT().Get(gomock.Any(), filter).Return(model.Userprofile{}, nil)
			},
			wantErr: true,
			errKey:  errkey.ErrUserprofileNotFound,
		},
		{
			name: "error - database query failed",
			id:   "profile-1",
			setupMock: func() {
				filter := shared.SingleFilter("profile-1", model.FieldID, model.TableName)
				mockRepo.EXPECT().Get(gomock.Any(), filter).Return(model.Userprofile{}, errors.New("db error"))
			},
			wantErr: true,
			errKey:  errkey.ErrDatabaseQuery,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := svc.Get(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				f, ok := err.(*failure.Failure)
				assert.True(t, ok)
				assert.Equal(t, tt.errKey, f.Key)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.id, result.ID)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	ctrl, mockRepo, cfg, mockS3, mockOtel := setup(t)
	defer ctrl.Finish()

	svc := service.New(mockRepo, cfg, mockS3, mockOtel)

	tests := []struct {
		name      string
		id        string
		req       dto.UpdateUserprofileRequest
		setupMock func()
		wantErr   bool
		errKey    errkey.ErrorKey
	}{
		{
			name: "success",
			id:   "profile-1",
			req:  dto.UpdateUserprofileRequest{Name: "Updated Name"},
			setupMock: func() {
				filter := shared.SingleFilter("profile-1", model.FieldID, model.TableName)
				mockRepo.EXPECT().Exist(gomock.Any(), filter).Return(true, nil)
				mockRepo.EXPECT().Update(gomock.Any(), gomock.Any(), filter).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - not found",
			id:   "profile-nonexistent",
			req:  dto.UpdateUserprofileRequest{Name: "Updated Name"},
			setupMock: func() {
				filter := shared.SingleFilter("profile-nonexistent", model.FieldID, model.TableName)
				mockRepo.EXPECT().Exist(gomock.Any(), filter).Return(false, nil)
			},
			wantErr: true,
			errKey:  errkey.ErrUserprofileNotFound,
		},
		{
			name: "error - database query failed on exist check",
			id:   "profile-1",
			req:  dto.UpdateUserprofileRequest{Name: "Updated Name"},
			setupMock: func() {
				filter := shared.SingleFilter("profile-1", model.FieldID, model.TableName)
				mockRepo.EXPECT().Exist(gomock.Any(), filter).Return(false, errors.New("db error"))
			},
			wantErr: true,
			errKey:  errkey.ErrDatabaseQuery,
		},
		{
			name: "error - database query failed on update",
			id:   "profile-1",
			req:  dto.UpdateUserprofileRequest{Name: "Updated Name"},
			setupMock: func() {
				filter := shared.SingleFilter("profile-1", model.FieldID, model.TableName)
				mockRepo.EXPECT().Exist(gomock.Any(), filter).Return(true, nil)
				mockRepo.EXPECT().Update(gomock.Any(), gomock.Any(), filter).Return(errors.New("db error"))
			},
			wantErr: true,
			errKey:  errkey.ErrUserprofileUpdateFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := svc.Update(context.Background(), tt.req, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				f, ok := err.(*failure.Failure)
				assert.True(t, ok)
				assert.Equal(t, tt.errKey, f.Key)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGeneratePresignedURL(t *testing.T) {
	ctrl, mockRepo, cfg, mockS3, mockOtel := setup(t)
	defer ctrl.Finish()

	cfg.External.S3.BucketName = "test-bucket"

	svc := service.New(mockRepo, cfg, mockS3, mockOtel)

	tests := []struct {
		name      string
		userID    string
		req       dto.GeneratePresignedURLRequest
		setupMock func()
		wantErr   bool
	}{
		{
			name:   "success",
			userID: "profile-1",
			req: dto.GeneratePresignedURLRequest{
				FileName:    "avatar.jpg",
				ContentType: "image/jpeg",
			},
			setupMock: func() {
				mockS3.EXPECT().GetPresignedUploadURL(gomock.Any(), "test-bucket", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("https://s3.example.com/upload", nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			_, err := svc.GeneratePresignedURL(context.Background(), tt.req, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
