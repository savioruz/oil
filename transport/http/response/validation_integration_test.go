package response_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"oil/shared/errkey"
	"oil/shared/failure"
	"oil/transport/http/response"

	"github.com/stretchr/testify/assert"
	"net/http/httptest"
)

// TestValidationErrorExamples demonstrates the actual JSON responses for validation errors
func TestValidationErrorExamples(t *testing.T) {
	t.Run("Single field validation error - required field", func(t *testing.T) {
		valErr := &failure.ValidationError{
			Code: http.StatusUnprocessableEntity,
			Fields: []failure.ValidationFieldError{
				{
					Field:   "title",
					Message: "Title is required",
					Key:     errkey.ErrorKey("validation.required.title"),
					Param:   "",
				},
			},
		}

		recorder := httptest.NewRecorder()
		response.WithError(recorder, valErr)

		// Parse and pretty print the response
		var result map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &result)
		prettyJSON, _ := json.MarshalIndent(result, "", "  ")

		fmt.Println("\n=== Missing required field (title) ===")
		fmt.Println(string(prettyJSON))

		assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
		assert.Equal(t, "validation.required.title", result["error"])
	})

	t.Run("Single field validation error - invalid URL", func(t *testing.T) {
		valErr := &failure.ValidationError{
			Code: http.StatusUnprocessableEntity,
			Fields: []failure.ValidationFieldError{
				{
					Field:   "images[0]",
					Message: "Images[0] must be a valid URL",
					Key:     errkey.ErrorKey("validation.url.images"),
					Param:   "",
				},
			},
		}

		recorder := httptest.NewRecorder()
		response.WithError(recorder, valErr)

		var result map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &result)
		prettyJSON, _ := json.MarshalIndent(result, "", "  ")

		fmt.Println("\n=== Invalid URL in array ===")
		fmt.Println(string(prettyJSON))

		assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
		assert.Equal(t, "validation.url.images", result["error"])
	})

	t.Run("Single field validation error - min length", func(t *testing.T) {
		valErr := &failure.ValidationError{
			Code: http.StatusUnprocessableEntity,
			Fields: []failure.ValidationFieldError{
				{
					Field:   "title",
					Message: "Title must be greater than or equal to 3",
					Key:     errkey.ErrorKey("validation.min.title"),
					Param:   "3",
				},
			},
		}

		recorder := httptest.NewRecorder()
		response.WithError(recorder, valErr)

		var result map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &result)
		prettyJSON, _ := json.MarshalIndent(result, "", "  ")

		fmt.Println("\n=== Min length validation ===")
		fmt.Println(string(prettyJSON))

		assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
		assert.Equal(t, "validation.min.title", result["error"])
	})

	t.Run("Multiple field validation errors - returns first", func(t *testing.T) {
		valErr := &failure.ValidationError{
			Code: http.StatusUnprocessableEntity,
			Fields: []failure.ValidationFieldError{
				{
					Field:   "title",
					Message: "Title is required",
					Key:     errkey.ErrorKey("validation.required.title"),
					Param:   "",
				},
				{
					Field:   "images",
					Message: "Images is required",
					Key:     errkey.ErrorKey("validation.required.images"),
					Param:   "",
				},
			},
		}

		recorder := httptest.NewRecorder()
		response.WithError(recorder, valErr)

		var result map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &result)
		prettyJSON, _ := json.MarshalIndent(result, "", "  ")

		fmt.Println("\n=== Multiple validation errors (returns first) ===")
		fmt.Println(string(prettyJSON))

		assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
		// Should return the FIRST field's error key
		assert.Equal(t, "validation.required.title", result["error"])
	})

	t.Run("Non-validation error - gallery not found", func(t *testing.T) {
		err := failure.NotFoundWithKey(errkey.ErrGalleryNotFound, "gallery not found")

		recorder := httptest.NewRecorder()
		response.WithError(recorder, err)

		var result map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &result)
		prettyJSON, _ := json.MarshalIndent(result, "", "  ")

		fmt.Println("\n=== Non-validation error (gallery not found) ===")
		fmt.Println(string(prettyJSON))

		assert.Equal(t, http.StatusNotFound, recorder.Code)
		assert.Equal(t, "gallery.not_found", result["error"])
	})
}
