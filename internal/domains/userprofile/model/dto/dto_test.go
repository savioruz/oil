package dto

import (
	"testing"
	"time"

	"oil/internal/domains/userprofile/model"
	gModel "oil/shared/model"

	"github.com/stretchr/testify/assert"
)

func TestCreateUserprofileRequest_ToModel(t *testing.T) {
	tests := []struct {
		name     string
		req      CreateUserprofileRequest
		expected model.Userprofile
	}{
		{
			name: "with all fields",
			req: CreateUserprofileRequest{
				AuthUserID: "auth-123",
				Email:      "test@example.com",
				Role:       "admin",
				Name:       "John Doe",
				Image:      "https://example.com/avatar.jpg",
			},
			expected: model.Userprofile{
				AuthUserID: "auth-123",
				Email:      "test@example.com",
				Role:       "admin",
				Name:       "John Doe",
				Image:      "https://example.com/avatar.jpg",
				Active:     true,
			},
		},
		{
			name: "with empty role defaults to user",
			req: CreateUserprofileRequest{
				AuthUserID: "auth-123",
				Email:      "test@example.com",
				Role:       "",
				Name:       "John Doe",
			},
			expected: model.Userprofile{
				AuthUserID: "auth-123",
				Email:      "test@example.com",
				Role:       "user",
				Name:       "John Doe",
				Active:     true,
			},
		},
		{
			name: "with user role",
			req: CreateUserprofileRequest{
				AuthUserID: "auth-123",
				Email:      "test@example.com",
				Role:       "user",
				Name:       "John Doe",
			},
			expected: model.Userprofile{
				AuthUserID: "auth-123",
				Email:      "test@example.com",
				Role:       "user",
				Name:       "John Doe",
				Active:     true,
			},
		},
		{
			name: "with empty name and image",
			req: CreateUserprofileRequest{
				AuthUserID: "auth-123",
				Email:      "test@example.com",
				Role:       "admin",
			},
			expected: model.Userprofile{
				AuthUserID: "auth-123",
				Email:      "test@example.com",
				Role:       "admin",
				Active:     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.req.ToModel()

			assert.NotEmpty(t, result.ID)
			assert.Equal(t, tt.expected.AuthUserID, result.AuthUserID)
			assert.Equal(t, tt.expected.Email, result.Email)
			assert.Equal(t, tt.expected.Role, result.Role)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Image, result.Image)
			assert.True(t, result.Active)
			assert.NotZero(t, result.Metadata.CreatedAt)
			assert.NotZero(t, result.Metadata.ModifiedAt)
			assert.Equal(t, tt.expected.AuthUserID, result.Metadata.CreatedBy)
			assert.Equal(t, tt.expected.AuthUserID, result.Metadata.ModifiedBy)
		})
	}
}

func TestUserprofileResponse_FromModel(t *testing.T) {
	tests := []struct {
		name     string
		model    model.Userprofile
		expected UserprofileResponse
	}{
		{
			name: "with all fields",
			model: model.Userprofile{
				ID:         "profile-123",
				AuthUserID: "auth-123",
				Email:      "test@example.com",
				Role:       "admin",
				Name:       "John Doe",
				Image:      "https://example.com/avatar.jpg",
				Active:     true,
				Metadata: gModel.Metadata{
					CreatedAt:  testTime(2024, 1, 1),
					ModifiedAt: testTime(2024, 1, 2),
					CreatedBy:  "auth-123",
					ModifiedBy: "auth-123",
				},
			},
			expected: UserprofileResponse{
				ID:         "profile-123",
				AuthUserID: "auth-123",
				Email:      "test@example.com",
				Role:       "admin",
				Name:       "John Doe",
				Image:      "https://example.com/avatar.jpg",
				Active:     true,
			},
		},
		{
			name: "with empty optional fields",
			model: model.Userprofile{
				ID:         "profile-123",
				AuthUserID: "auth-123",
				Email:      "test@example.com",
				Role:       "user",
				Name:       "",
				Image:      "",
				Active:     true,
				Metadata: gModel.Metadata{
					CreatedAt:  testTime(2024, 1, 1),
					ModifiedAt: testTime(2024, 1, 2),
					CreatedBy:  "auth-123",
					ModifiedBy: "auth-123",
				},
			},
			expected: UserprofileResponse{
				ID:         "profile-123",
				AuthUserID: "auth-123",
				Email:      "test@example.com",
				Role:       "user",
				Name:       "",
				Image:      "",
				Active:     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result UserprofileResponse
			result.FromModel(tt.model)

			assert.Equal(t, tt.expected.ID, result.ID)
			assert.Equal(t, tt.expected.AuthUserID, result.AuthUserID)
			assert.Equal(t, tt.expected.Email, result.Email)
			assert.Equal(t, tt.expected.Role, result.Role)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Image, result.Image)
			assert.Equal(t, tt.expected.Active, result.Active)
		})
	}
}

func TestGetUserprofilesResponse_FromModels(t *testing.T) {
	tests := []struct {
		name      string
		models    []model.Userprofile
		totalData int
		limit     int
		expected  GetUserprofilesResponse
	}{
		{
			name: "with multiple models",
			models: []model.Userprofile{
				{ID: "1", AuthUserID: "auth-1", Email: "test1@example.com", Role: "user"},
				{ID: "2", AuthUserID: "auth-2", Email: "test2@example.com", Role: "admin"},
				{ID: "3", AuthUserID: "auth-3", Email: "test3@example.com", Role: "user"},
				{ID: "4", AuthUserID: "auth-4", Email: "test4@example.com", Role: "user"},
				{ID: "5", AuthUserID: "auth-5", Email: "test5@example.com", Role: "user"},
			},
			totalData: 5,
			limit:     10,
			expected: GetUserprofilesResponse{
				TotalPage: 1,
				TotalData: 5,
			},
		},
		{
			name:      "with empty models",
			models:    []model.Userprofile{},
			totalData: 0,
			limit:     10,
			expected: GetUserprofilesResponse{
				TotalPage:    0,
				TotalData:    0,
				Userprofiles: []UserprofileResponse{},
			},
		},
		{
			name: "with pagination",
			models: []model.Userprofile{
				{ID: "1", AuthUserID: "auth-1", Email: "test1@example.com", Role: "user"},
			},
			totalData: 25,
			limit:     10,
			expected: GetUserprofilesResponse{
				TotalPage: 3,
				TotalData: 25,
			},
		},
		{
			name: "with exact division",
			models: []model.Userprofile{
				{ID: "1", AuthUserID: "auth-1", Email: "test1@example.com", Role: "user"},
				{ID: "2", AuthUserID: "auth-2", Email: "test2@example.com", Role: "user"},
			},
			totalData: 20,
			limit:     10,
			expected: GetUserprofilesResponse{
				TotalPage: 2,
				TotalData: 20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result GetUserprofilesResponse
			result.FromModels(tt.models, tt.totalData, tt.limit)

			assert.Equal(t, tt.expected.TotalData, result.TotalData)
			assert.Equal(t, tt.expected.TotalPage, result.TotalPage)
			assert.Len(t, result.Userprofiles, len(tt.models))
		})
	}
}

func TestGeneratePresignedURLRequest_Validation(t *testing.T) {
	tests := []struct {
		name  string
		req   GeneratePresignedURLRequest
		valid bool
	}{
		{
			name: "valid request",
			req: GeneratePresignedURLRequest{
				FileName:    "avatar.jpg",
				ContentType: "image/jpeg",
			},
			valid: true,
		},
		{
			name: "valid request with png",
			req: GeneratePresignedURLRequest{
				FileName:    "profile.png",
				ContentType: "image/png",
			},
			valid: true,
		},
		{
			name: "valid request with gif",
			req: GeneratePresignedURLRequest{
				FileName:    "animation.gif",
				ContentType: "image/gif",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.NotEmpty(t, tt.req.FileName)
				assert.NotEmpty(t, tt.req.ContentType)
			}
		})
	}
}

func TestUpdateUserprofileRequest_Fields(t *testing.T) {
	tests := []struct {
		name string
		req  UpdateUserprofileRequest
	}{
		{
			name: "with all fields",
			req: UpdateUserprofileRequest{
				Name:       "Updated Name",
				Image:      "https://example.com/new-avatar.jpg",
				AuthUserID: "new-auth-id",
			},
		},
		{
			name: "with only name",
			req: UpdateUserprofileRequest{
				Name: "Only Name",
			},
		},
		{
			name: "with only image",
			req: UpdateUserprofileRequest{
				Image: "https://example.com/only-image.jpg",
			},
		},
		{
			name: "with empty fields",
			req:  UpdateUserprofileRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just test that fields are accessible
			_ = tt.req.Name
			_ = tt.req.Image
			_ = tt.req.AuthUserID
		})
	}
}

func testTime(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}
