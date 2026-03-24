package gallery_test

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"oil/config"
	"oil/infras/otel/mocks"
	postgresMocks "oil/infras/postgres/mocks"
	s3Mocks "oil/infras/s3/mocks"
	"oil/internal/domains/gallery/model/dto"
	"oil/internal/domains/gallery/repository"
	"oil/internal/domains/gallery/service"
	"oil/internal/handlers/gallery"
	cacheMocks "oil/shared/cache/mocks"
	"oil/shared/constant"
	"oil/transport/http/response"
)

func setup(t *testing.T, ctrl *gomock.Controller) (*httptest.Server, sqlmock.Sqlmock, *cacheMocks.MockRedisCache, *s3Mocks.MockS3) {
	mux := chi.NewRouter()
	otel := mocks.NewOtel()
	cfg := &config.Config{}
	cfg.Cache.TTL = 300

	_, sqlMock, sqlConn := postgresMocks.SetupPostgresConnection(t)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockS3 := s3Mocks.NewMockS3(ctrl)

	repo := repository.New(sqlConn, otel)
	svc := service.New(repo, cfg, mockCache, otel, mockS3)
	handler := gallery.New(svc, mockS3, otel)

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

	return ts, sqlMock, mockCache, mockS3
}

func TestCreateGallery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, sqlMock, mockCache, _ := setup(t, ctrl)
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

	validBody := dto.CreateGalleryRequest{
		Title:       "Test Gallery",
		Description: "This is a test gallery",
		Images:      []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg"},
	}

	t.Run("Error: invalid request - missing title", func(t *testing.T) {
		defer resetResponse()

		invalidBody := map[string]interface{}{
			"description": "This is a test gallery",
			"images":      []string{"https://example.com/image1.jpg"},
		}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Post("/galleries")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: invalid request - title too short", func(t *testing.T) {
		defer resetResponse()

		invalidBody := map[string]interface{}{
			"title":       "ab",
			"description": "This is a test gallery",
			"images":      []string{"https://example.com/image1.jpg"},
		}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Post("/galleries")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: invalid request - missing images", func(t *testing.T) {
		defer resetResponse()

		invalidBody := map[string]interface{}{
			"title":       "Test Gallery",
			"description": "This is a test gallery",
		}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Post("/galleries")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: invalid request - invalid image URL", func(t *testing.T) {
		defer resetResponse()

		invalidBody := map[string]interface{}{
			"title":       "Test Gallery",
			"description": "This is a test gallery",
			"images":      []string{"not-a-url"},
		}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Post("/galleries")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: failed to insert gallery", func(t *testing.T) {
		defer resetResponse()

		sqlMock.ExpectExec(regexp.QuoteMeta("INSERT INTO galleries")).
			WillReturnError(fmt.Errorf("insert error"))

		resp, err := getClient().
			SetBody(validBody).
			Post("/galleries")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Success: create gallery", func(t *testing.T) {
		defer resetResponse()

		sqlMock.ExpectExec(regexp.QuoteMeta("INSERT INTO galleries")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().
			SetBody(validBody).
			Post("/galleries")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusCreated, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseMessage.Message)
		assert.Equal(t, "Gallery created successfully", *responseMessage.Message)
	})
}

func TestGetGalleries(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, sqlMock, mockCache, _ := setup(t, ctrl)
	defer ts.Close()

	var responseData response.Data[dto.GetGalleriesResponse]
	var responseErr response.Error

	resetResponse := func() {
		responseData = response.Data[dto.GetGalleriesResponse]{}
		responseErr = response.Error{}
	}

	getClient := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseData).
			SetError(&responseErr)
	}

	t.Run("Error: failed to count galleries", func(t *testing.T) {
		defer resetResponse()

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT COUNT(galleries.id) FROM galleries")).
			ExpectQuery().
			WillReturnError(fmt.Errorf("count error"))

		resp, err := getClient().Get("/galleries")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: failed to get galleries", func(t *testing.T) {
		defer resetResponse()

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT COUNT(galleries.id) FROM galleries")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		sqlMock.ExpectPrepare("SELECT .* FROM galleries").
			ExpectQuery().
			WillReturnError(fmt.Errorf("query error"))

		resp, err := getClient().Get("/galleries")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Success: get all galleries", func(t *testing.T) {
		defer resetResponse()

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT COUNT(galleries.id) FROM galleries")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		now := time.Now()
		images1 := pq.StringArray{"https://example.com/image1.jpg", "https://example.com/image2.jpg"}
		images2 := pq.StringArray{"https://example.com/image3.jpg"}
		sqlMock.ExpectPrepare("SELECT .* FROM galleries").
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "images", "created_at", "modified_at", "created_by", "modified_by"}).
				AddRow("id1", "Gallery 1", "Description 1", images1, now, now, "user1", "user1").
				AddRow("id2", "Gallery 2", "Description 2", images2, now, now, "user1", "user1"))

		mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().Get("/galleries")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseData.Data)
		assert.Equal(t, 2, len(responseData.Data.Galleries))
		assert.Equal(t, "Gallery 1", responseData.Data.Galleries[0].Title)
		assert.Equal(t, "Gallery 2", responseData.Data.Galleries[1].Title)
	})

	t.Run("Success: get galleries with pagination", func(t *testing.T) {
		defer resetResponse()

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT COUNT(galleries.id) FROM galleries")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

		now := time.Now()
		images := pq.StringArray{"https://example.com/image1.jpg"}
		sqlMock.ExpectPrepare("SELECT .* FROM galleries").
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "images", "created_at", "modified_at", "created_by", "modified_by"}).
				AddRow("id1", "Gallery 1", "Description 1", images, now, now, "user1", "user1"))

		mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().
			SetQueryParam("page", "2").
			SetQueryParam("limit", "10").
			Get("/galleries")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseData.Data)
		assert.Equal(t, 25, responseData.Data.TotalData)
		assert.Equal(t, 3, responseData.Data.TotalPage)
	})

	t.Run("Success: filter by title", func(t *testing.T) {
		defer resetResponse()

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))
		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT COUNT(galleries.id) FROM galleries")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		now := time.Now()
		images := pq.StringArray{"https://example.com/vacation.jpg"}
		sqlMock.ExpectPrepare("SELECT .* FROM galleries").
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "images", "created_at", "modified_at", "created_by", "modified_by"}).
				AddRow("id1", "Vacation Photos", "Summer vacation", images, now, now, "user1", "user1"))

		mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().
			SetQueryParam("title", "Vacation").
			Get("/galleries")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseData.Data)
		assert.Equal(t, 1, len(responseData.Data.Galleries))
		assert.Contains(t, responseData.Data.Galleries[0].Title, "Vacation")
	})

	t.Run("Success: cache hit", func(t *testing.T) {
		defer resetResponse()

		cachedResponse := dto.GetGalleriesResponse{
			Galleries: []dto.GalleryResponse{
				{
					ID:          "id1",
					Title:       "Cached Gallery",
					Description: "From cache",
					Images:      []string{"https://example.com/cached.jpg"},
				},
			},
			TotalPage: 1,
			TotalData: 1,
		}

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, key string, dest interface{}) error {
				*dest.(*dto.GetGalleriesResponse) = cachedResponse
				return nil
			})

		resp, err := getClient().Get("/galleries")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.Equal(t, 1, len(responseData.Data.Galleries))
		assert.Equal(t, "Cached Gallery", responseData.Data.Galleries[0].Title)
	})
}

func TestGetGalleryByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, sqlMock, mockCache, _ := setup(t, ctrl)
	defer ts.Close()

	var responseData response.Data[dto.GalleryResponse]
	var responseErr response.Error

	resetResponse := func() {
		responseData = response.Data[dto.GalleryResponse]{}
		responseErr = response.Error{}
	}

	getClient := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseData).
			SetError(&responseErr)
	}

	t.Run("Error: gallery not found", func(t *testing.T) {
		defer resetResponse()

		galleryID := "non-existent-id"

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		sqlMock.ExpectPrepare("SELECT .* FROM galleries WHERE \\(galleries\\.id = \\?\\)").
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "images", "created_at", "modified_at", "created_by", "modified_by"}))

		resp, err := getClient().Get("/galleries/" + galleryID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusNotFound, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: database query error", func(t *testing.T) {
		defer resetResponse()

		galleryID := "test-id"

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		sqlMock.ExpectPrepare("SELECT .* FROM galleries WHERE \\(galleries\\.id = \\?\\)").
			ExpectQuery().
			WillReturnError(fmt.Errorf("database error"))

		resp, err := getClient().Get("/galleries/" + galleryID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Success: get gallery by ID", func(t *testing.T) {
		defer resetResponse()

		galleryID := "test-gallery-id"

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache miss"))

		now := time.Now()
		images := pq.StringArray{"https://example.com/test1.jpg", "https://example.com/test2.jpg"}
		sqlMock.ExpectPrepare("SELECT .* FROM galleries WHERE \\(galleries\\.id = \\?\\)").
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "images", "created_at", "modified_at", "created_by", "modified_by"}).
				AddRow(galleryID, "Test Gallery", "Test Description", images, now, now, "user1", "user1"))

		mockCache.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().Get("/galleries/" + galleryID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseData.Data)
		if responseData.Data != nil {
			assert.Equal(t, galleryID, responseData.Data.ID)
			assert.Equal(t, "Test Gallery", responseData.Data.Title)
			assert.Equal(t, "Test Description", responseData.Data.Description)
			assert.Equal(t, 2, len(responseData.Data.Images))
		}
	})

	t.Run("Success: cache hit", func(t *testing.T) {
		defer resetResponse()

		galleryID := "cached-gallery-id"

		cachedGallery := dto.GalleryResponse{
			ID:          galleryID,
			Title:       "Cached Gallery",
			Description: "From cache",
			Images:      []string{"https://example.com/cached.jpg"},
		}

		mockCache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, key string, dest interface{}) error {
				*dest.(*dto.GalleryResponse) = cachedGallery
				return nil
			})

		resp, err := getClient().Get("/galleries/" + galleryID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseData.Data)
		if responseData.Data != nil {
			assert.Equal(t, "Cached Gallery", responseData.Data.Title)
		}
	})
}

func TestUpdateGallery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, sqlMock, mockCache, _ := setup(t, ctrl)
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

	validBody := dto.UpdateGalleryRequest{
		Title:       "Updated Title",
		Description: "Updated description",
		Images:      []string{"https://example.com/updated.jpg"},
	}

	t.Run("Error: invalid request - title too short", func(t *testing.T) {
		defer resetResponse()

		invalidBody := map[string]interface{}{
			"title": "ab",
		}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Patch("/galleries/test-id")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: gallery not found", func(t *testing.T) {
		defer resetResponse()

		galleryID := "non-existent-id"

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM galleries WHERE (galleries.id = ?) )")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		resp, err := getClient().
			SetBody(validBody).
			Patch("/galleries/" + galleryID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusNotFound, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: failed to check existence", func(t *testing.T) {
		defer resetResponse()

		galleryID := "test-id"

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM galleries WHERE (galleries.id = ?) )")).
			ExpectQuery().
			WillReturnError(fmt.Errorf("db error"))

		resp, err := getClient().
			SetBody(validBody).
			Patch("/galleries/" + galleryID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: failed to update gallery", func(t *testing.T) {
		defer resetResponse()

		galleryID := "test-id"

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM galleries WHERE (galleries.id = ?) )")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		sqlMock.ExpectExec(regexp.QuoteMeta("UPDATE galleries SET")).
			WillReturnError(fmt.Errorf("update error"))

		resp, err := getClient().
			SetBody(validBody).
			Patch("/galleries/" + galleryID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Success: update gallery", func(t *testing.T) {
		defer resetResponse()

		galleryID := "test-id"

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM galleries WHERE (galleries.id = ?) )")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		sqlMock.ExpectExec(regexp.QuoteMeta("UPDATE galleries SET")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mockCache.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().
			SetBody(validBody).
			Patch("/galleries/" + galleryID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		if assert.NotNil(t, responseMessage.Message) {
			assert.Equal(t, "Gallery updated successfully", *responseMessage.Message)
		}
	})

	t.Run("Success: partial update - only title", func(t *testing.T) {
		defer resetResponse()

		galleryID := "test-id"
		partialBody := dto.UpdateGalleryRequest{
			Title: "Only Title Updated",
		}

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM galleries WHERE (galleries.id = ?) )")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		sqlMock.ExpectExec(regexp.QuoteMeta("UPDATE galleries SET")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mockCache.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().
			SetBody(partialBody).
			Patch("/galleries/" + galleryID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
	})
}

func TestDeleteGallery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, sqlMock, mockCache, mockS3 := setup(t, ctrl)
	defer ts.Close()

	var responseMessage response.Message
	var responseErr response.Error

	resetResponse := func() {
		responseMessage = response.Message{}
		responseErr = response.Error{}
	}

	getClient := func() *resty.Request {
		return resty.New().
			SetBaseURL(ts.URL + "/api").
			R().
			SetResult(&responseMessage).
			SetError(&responseErr)
	}

	t.Run("Error: gallery not found", func(t *testing.T) {
		defer resetResponse()

		galleryID := "non-existent-id"

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT ")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "images", "created_at", "modified_at", "created_by", "modified_by"}))

		resp, err := getClient().Delete("/galleries/" + galleryID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusNotFound, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: failed to get gallery", func(t *testing.T) {
		defer resetResponse()

		galleryID := "test-id"

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT ")).
			ExpectQuery().
			WillReturnError(fmt.Errorf("db error"))

		resp, err := getClient().Delete("/galleries/" + galleryID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Error: failed to delete gallery", func(t *testing.T) {
		defer resetResponse()

		galleryID := "test-id"

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT ")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "images", "created_at", "modified_at", "created_by", "modified_by"}).
				AddRow(galleryID, "Test Gallery", "Test Description", pq.StringArray{"https://example.com/image1.jpg"}, time.Now(), time.Now(), "user-id", "user-id"))

		sqlMock.ExpectExec(regexp.QuoteMeta("DELETE FROM galleries WHERE (galleries.id = ?)")).
			WillReturnError(fmt.Errorf("delete error"))

		resp, err := getClient().Delete("/galleries/" + galleryID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Success: delete gallery", func(t *testing.T) {
		defer resetResponse()

		galleryID := "test-id"

		sqlMock.ExpectPrepare(regexp.QuoteMeta("SELECT ")).
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "images", "created_at", "modified_at", "created_by", "modified_by"}).
				AddRow(galleryID, "Test Gallery", "Test Description", pq.StringArray{"https://example.com/image1.jpg"}, time.Now(), time.Now(), "user-id", "user-id"))

		sqlMock.ExpectExec(regexp.QuoteMeta("DELETE FROM galleries WHERE (galleries.id = ?)")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mockCache.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockCache.EXPECT().Clear(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		mockS3.EXPECT().GetObjectNameFromURL(gomock.Any(), gomock.Any()).Return("image1.jpg").AnyTimes()
		mockS3.EXPECT().DeleteFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		resp, err := getClient().Delete("/galleries/" + galleryID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		if assert.NotNil(t, responseMessage.Message) {
			assert.Equal(t, "Gallery deleted successfully", *responseMessage.Message)
		}
	})
}

func TestUploadImage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, _, _, mockS3 := setup(t, ctrl)
	defer ts.Close()

	var responseData response.Data[dto.UploadImageResponse]
	var responseErr response.Error

	resetResponse := func() {
		responseData = response.Data[dto.UploadImageResponse]{}
		responseErr = response.Error{}
	}

	createMultipartForm := func(filename, content, contentType string) (*bytes.Buffer, string) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", filename)
		part.Write([]byte(content))
		writer.Close()
		return body, writer.FormDataContentType()
	}

	t.Run("Error: missing file", func(t *testing.T) {
		defer resetResponse()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.Close()

		req, _ := http.NewRequest("POST", ts.URL+"/api/galleries/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Error: failed to upload to S3", func(t *testing.T) {
		defer resetResponse()

		mockS3.EXPECT().UploadFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("", fmt.Errorf("s3 upload error"))

		body, contentType := createMultipartForm("test.jpg", "fake image content", "image/jpeg")

		req, _ := http.NewRequest("POST", ts.URL+"/api/galleries/upload", body)
		req.Header.Set("Content-Type", contentType)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Success: upload image", func(t *testing.T) {
		defer resetResponse()

		mockS3.EXPECT().UploadFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("https://s3.amazonaws.com/bucket/test-image.jpg", nil)

		body, contentType := createMultipartForm("test.jpg", "fake image content", "image/jpeg")

		client := resty.New()
		resp, err := client.R().
			SetHeader("Content-Type", contentType).
			SetBody(body.Bytes()).
			SetResult(&responseData).
			SetError(&responseErr).
			Post(ts.URL + "/api/galleries/upload")

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseData.Data)
		if responseData.Data != nil {
			assert.NotEmpty(t, responseData.Data.URL)
			assert.NotEmpty(t, responseData.Data.FileName)
		}
	})
}

func TestDeleteImages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ts, _, _, mockS3 := setup(t, ctrl)
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

	t.Run("Error: invalid request - missing image URLs", func(t *testing.T) {
		defer resetResponse()

		invalidBody := map[string]interface{}{}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Delete("/galleries/images")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: invalid request - empty image URLs array", func(t *testing.T) {
		defer resetResponse()

		invalidBody := map[string]interface{}{
			"image_urls": []string{},
		}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Delete("/galleries/images")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: invalid request - invalid URL format", func(t *testing.T) {
		defer resetResponse()

		invalidBody := map[string]interface{}{
			"image_urls": []string{"not-a-url"},
		}

		resp, err := getClientForValidation().
			SetBody(invalidBody).
			Delete("/galleries/images")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())
		assert.NotEmpty(t, responseValidationErr.Errors)
	})

	t.Run("Error: failed to delete from S3", func(t *testing.T) {
		defer resetResponse()

		validBody := dto.DeleteImagesRequest{
			ImageURLs: []string{"https://s3.amazonaws.com/bucket/test1.jpg"},
		}

		mockS3.EXPECT().GetObjectNameFromURL(gomock.Any(), gomock.Any()).
			Return("test1.jpg")
		mockS3.EXPECT().DeleteFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(fmt.Errorf("s3 delete error"))

		resp, err := getClient().
			SetBody(validBody).
			Delete("/galleries/images")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.NotEmpty(t, responseErr.Error)
	})

	t.Run("Success: delete images", func(t *testing.T) {
		defer resetResponse()

		validBody := dto.DeleteImagesRequest{
			ImageURLs: []string{
				"https://s3.amazonaws.com/bucket/test1.jpg",
				"https://s3.amazonaws.com/bucket/test2.jpg",
			},
		}

		mockS3.EXPECT().GetObjectNameFromURL(gomock.Any(), gomock.Any()).
			Return("test1.jpg")
		mockS3.EXPECT().DeleteFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)
		mockS3.EXPECT().GetObjectNameFromURL(gomock.Any(), gomock.Any()).
			Return("test2.jpg")
		mockS3.EXPECT().DeleteFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)

		resp, err := getClient().
			SetBody(validBody).
			Delete("/galleries/images")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Empty(t, responseErr.Error)
		assert.NotNil(t, responseMessage.Message)
		assert.Equal(t, "Images deleted successfully", *responseMessage.Message)
	})
}
