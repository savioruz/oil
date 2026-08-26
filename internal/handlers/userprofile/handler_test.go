package userprofile_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/savioruz/oil/config"
	"github.com/savioruz/oil/infras/otel/mocks"
	postgresMocks "github.com/savioruz/oil/infras/postgres/mocks"
	s3mocks "github.com/savioruz/oil/infras/s3/mocks"
	"github.com/savioruz/oil/internal/handlers/userprofile"
	"github.com/savioruz/oil/internal/modules/userprofile/model/dto"
	"github.com/savioruz/oil/internal/modules/userprofile/repository"
	"github.com/savioruz/oil/internal/modules/userprofile/service"
	"github.com/savioruz/oil/shared/constant"
	"github.com/savioruz/oil/transport/http/response"
)

func setup(t *testing.T, ctrl *gomock.Controller) (*httptest.Server, sqlmock.Sqlmock, *s3mocks.MockS3) {
	mux := chi.NewRouter()
	otel := mocks.NewOtel()
	cfg := &config.Config{}
	cfg.External.S3.BucketName = "test-bucket"

	_, sqlMock, sqlConn := postgresMocks.SetupPostgresConnection(t)

	repo := repository.New(sqlConn, otel)
	mockS3 := s3mocks.NewMockS3(ctrl)
	svc := service.New(repo, cfg, mockS3, otel)
	handler := userprofile.New(svc, otel)

	mux.Route("/api", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.WithValue(r.Context(), constant.ContextKeyUserID, "test-user-id")
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})
		handler.Router(r)
	})

	ts := httptest.NewServer(mux)

	return ts, sqlMock, mockS3
}

func TestGetMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, sqlMock, _ := setup(t, ctrl)
	defer ts.Close()

	var responseData response.Data[dto.UserprofileResponse]
	var responseErr response.Error

	resetResponse := func() {
		responseData = response.Data[dto.UserprofileResponse]{}
		responseErr = response.Error{}
	}

	getClient := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseData).
			SetError(&responseErr)
	}

	t.Run("Error: userprofile not found", func(t *testing.T) {
		defer resetResponse()

		sqlMock.ExpectQuery("SELECT .* FROM user_profiles").
			WillReturnRows(sqlmock.NewRows([]string{"id", "auth_user_id", "email", "role", "name", "image", "active", "created_at", "modified_at", "created_by", "modified_by"}))

		resp, err := getClient().Get("/users")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusNotFound, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: database query error", func(t *testing.T) {
		defer resetResponse()

		sqlMock.ExpectQuery("SELECT .* FROM user_profiles").
			WillReturnError(fmt.Errorf("database error"))

		resp, err := getClient().Get("/users")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Success: get userprofile", func(t *testing.T) {
		defer resetResponse()

		now := time.Now()
		sqlMock.ExpectQuery("SELECT .* FROM user_profiles").
			WillReturnRows(sqlmock.NewRows([]string{"id", "auth_user_id", "email", "role", "name", "image", "active", "created_at", "modified_at", "created_by", "modified_by"}).
				AddRow("user-id-1", "auth-123", "test@example.com", "user", "Test User", "https://example.com/image.jpg", true, now, now, "auth-123", "auth-123"))

		resp, err := getClient().Get("/users")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.Equal(t, "user-id-1", responseData.Data.ID)
		assert.Equal(t, "test@example.com", responseData.Data.Email)
		assert.Equal(t, "Test User", responseData.Data.Name)
	})
}

func TestUpdateMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, sqlMock, _ := setup(t, ctrl)
	defer ts.Close()

	var responseMessage response.Message
	var responseErr response.Error
	var responseValidationErr response.ValidationErrors

	resetResponse := func() {
		responseMessage = response.Message{}
		responseErr = response.Error{}
		responseValidationErr = response.ValidationErrors{}
	}

	getClient := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseMessage).
			SetError(&responseErr)
	}

	getClientForValidation := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseMessage).
			SetError(&responseValidationErr)
	}

	validBody := dto.UpdateUserprofileRequest{
		Name:  "Updated Name",
		Image: "https://example.com/new-image.jpg",
	}

	t.Run("Error: invalid request - name too long", func(t *testing.T) {
		defer resetResponse()

		longName := ""
		for i := 0; i < 256; i++ {
			longName += "a"
		}

		invalidBody := map[string]interface{}{
			"name": longName,
		}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Patch("/users")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: userprofile not found", func(t *testing.T) {
		defer resetResponse()

		sqlMock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM user_profiles WHERE (user_profiles.id = ?) )")).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		resp, err := getClient().
			SetBody(validBody).
			Patch("/users")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusNotFound, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: failed to check existence", func(t *testing.T) {
		defer resetResponse()

		sqlMock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM user_profiles WHERE (user_profiles.id = ?) )")).
			WillReturnError(fmt.Errorf("db error"))

		resp, err := getClient().
			SetBody(validBody).
			Patch("/users")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: failed to update userprofile", func(t *testing.T) {
		defer resetResponse()

		sqlMock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM user_profiles WHERE (user_profiles.id = ?) )")).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		sqlMock.ExpectExec(regexp.QuoteMeta("UPDATE user_profiles SET")).
			WillReturnError(fmt.Errorf("update error"))

		resp, err := getClient().
			SetBody(validBody).
			Patch("/users")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Success: update userprofile", func(t *testing.T) {
		defer resetResponse()

		sqlMock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM user_profiles WHERE (user_profiles.id = ?) )")).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		sqlMock.ExpectExec(regexp.QuoteMeta("UPDATE user_profiles SET")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		resp, err := getClient().
			SetBody(validBody).
			Patch("/users")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotEmpty(t, responseMessage.Message)
		assert.Equal(t, "Userprofile updated successfully", responseMessage.Message)
	})

	t.Run("Success: partial update - only name", func(t *testing.T) {
		defer resetResponse()

		partialBody := map[string]interface{}{
			"name": "New Name",
		}

		sqlMock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM user_profiles WHERE (user_profiles.id = ?) )")).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		sqlMock.ExpectExec(regexp.QuoteMeta("UPDATE user_profiles SET")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		resp, err := getClient().
			SetBody(partialBody).
			Patch("/users")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
	})
}

func TestGeneratePresignedURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, sqlMock, mockS3 := setup(t, ctrl)
	defer ts.Close()

	var responseData response.Data[dto.GeneratePresignedURLResponse]
	var responseErr response.Error
	var responseValidationErr response.ValidationErrors

	resetResponse := func() {
		responseData = response.Data[dto.GeneratePresignedURLResponse]{}
		responseErr = response.Error{}
		responseValidationErr = response.ValidationErrors{}
	}

	getClient := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseData).
			SetError(&responseErr)
	}

	getClientForValidation := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseData).
			SetError(&responseValidationErr)
	}

	t.Run("Error: invalid request - missing file_name", func(t *testing.T) {
		defer resetResponse()

		invalidBody := map[string]interface{}{
			"content_type": "image/jpeg",
		}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Post("/users/presigned-url")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: invalid request - missing content_type", func(t *testing.T) {
		defer resetResponse()

		invalidBody := map[string]interface{}{
			"file_name": "test.jpg",
		}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Post("/users/presigned-url")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: failed to generate presigned URL", func(t *testing.T) {
		defer resetResponse()

		validBody := map[string]interface{}{
			"file_name":    "test.jpg",
			"content_type": "image/jpeg",
		}

		sqlMock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM user_profiles WHERE (user_profiles.id = ?) )")).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		mockS3.EXPECT().GetPresignedUploadURL(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("", fmt.Errorf("s3 error"))

		resp, err := getClient().
			SetBody(validBody).
			Post("/users/presigned-url")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
	})

	t.Run("Success: generate presigned URL", func(t *testing.T) {
		defer resetResponse()

		validBody := map[string]interface{}{
			"file_name":    "test.jpg",
			"content_type": "image/jpeg",
		}

		sqlMock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM user_profiles WHERE (user_profiles.id = ?) )")).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		mockS3.EXPECT().GetPresignedUploadURL(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("https://s3.example.com/upload?token=abc", nil)

		resp, err := getClient().
			SetBody(validBody).
			Post("/users/presigned-url")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotEmpty(t, responseData.Data.UploadURL)
		assert.NotEmpty(t, responseData.Data.FileKey)
	})
}
