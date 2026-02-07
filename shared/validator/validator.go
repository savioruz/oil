package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	val "github.com/go-playground/validator/v10"
	"io"
	"mime/multipart"
	"net/http"
	"oil/config"
	"oil/shared/base64"
	"oil/shared/constant"
	"oil/shared/errkey"
	"oil/shared/failure"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

var validate *val.Validate

func registerMimetypeValidation(field val.FieldLevel) bool {
	var contentType string

	if file, ok := field.Field().Interface().(multipart.FileHeader); ok {
		contentType = file.Header.Get(constant.RequestHeaderContentType)
	} else if str, ok := field.Field().Interface().(string); ok {
		contentType = base64.GetContentType(str)

		if contentType == "" {
			return false
		}
	}

	allowedTypes := strings.Split(field.Param(), " ")

	return slices.Contains(allowedTypes, contentType)
}

func registerFileSizeValidation(field val.FieldLevel) bool {
	fileSize := 0
	if file, ok := field.Field().Interface().(multipart.FileHeader); ok {
		fileSize = int(file.Size)
	} else if str, ok := field.Field().Interface().(string); ok {
		fileSize = len(str)
	}

	maxSizeMB, err := strconv.ParseFloat(field.Param(), 64)
	if err != nil {
		return false
	}

	bytesConversion := 1024.0
	maxSizeBytes := int(maxSizeMB * bytesConversion * bytesConversion)

	return fileSize <= maxSizeBytes
}

func init() {
	cfg := config.Get()

	validate = val.New(val.WithRequiredStructEnabled())
	err := validate.RegisterValidation("oil", func(fl val.FieldLevel) bool {
		method := fl.Field().MethodByName("Validate")
		if method.IsValid() {
			result := method.Call([]reflect.Value{reflect.ValueOf(cfg)})

			return result[0].Interface() == nil
		}

		return false
	})

	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("empty", func(fl val.FieldLevel) bool {
		empty := fl.Field().IsZero()

		return empty
	})

	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("mimetypes", registerMimetypeValidation)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("maxfilesize", registerFileSizeValidation)
	if err != nil {
		panic(err)
	}
}

// Validate reads from the given io.Reader into the given struct, and then performs validation
// on the struct using the validator package. If the struct is invalid according to the
// validation rules, an error is returned. Otherwise, nil is returned.
// https://github.com/go-playground/validator
func Validate[T any](r io.Reader, data *T) error {
	decoder := json.NewDecoder(r)
	err := decoder.Decode(data)

	if err != nil {
		return failure.UnprocessableEntity(fmt.Errorf("failed to decode request body: %w", err)) //nolint:wrapcheck
	}

	return ValidateStruct(data)
}

func ValidateStruct[T any](data *T) error {
	err := validate.Struct(data)

	if err != nil {
		return buildValidationError(err)
	}

	return nil
}

func ValidateVar(field any, tag string) error {
	err := validate.Var(field, tag)

	if err != nil {
		return buildValidationError(err)
	}

	return nil
}

// buildValidationError converts validator errors into structured ValidationError
func buildValidationError(err error) error {
	var valErrors val.ValidationErrors
	if !errors.As(err, &valErrors) {
		// Not a validation error, return as unprocessable entity
		return failure.UnprocessableEntity(err) //nolint:wrapcheck
	}

	validationErr := &failure.ValidationError{
		Code:   http.StatusUnprocessableEntity,
		Fields: make([]failure.ValidationFieldError, 0, len(valErrors)),
	}

	for _, valErr := range valErrors {
		fieldName := valErr.Field()
		tag := valErr.Tag()
		param := valErr.Param()

		// Build field path (handle array indices)
		fieldPath := buildFieldPath(valErr)

		// Generate error key (e.g., "validation.required.title")
		errorKey := errkey.FormatFieldError(tag, fieldName)

		// Generate human-readable message
		message := generateFieldMessage(fieldName, tag, param)

		validationErr.Fields = append(validationErr.Fields, failure.ValidationFieldError{
			Field:   fieldPath,
			Message: message,
			Key:     errorKey,
			Param:   param,
		})
	}

	return validationErr
}

// buildFieldPath constructs the full field path including array indices
// e.g., "Images[0]" -> "images[0]", "Title" -> "title"
func buildFieldPath(valErr val.FieldError) string {
	namespace := valErr.Namespace()
	structName := valErr.StructNamespace()

	// Remove struct name prefix to get just the field path
	fieldPath := strings.TrimPrefix(namespace, structName)
	fieldPath = strings.TrimPrefix(fieldPath, ".")

	// Convert to snake_case but preserve array indices
	result := ""
	inBracket := false
	for i, r := range fieldPath {
		if r == '[' {
			inBracket = true
			result += string(r)
		} else if r == ']' {
			inBracket = false
			result += string(r)
		} else if inBracket {
			result += string(r)
		} else if r >= 'A' && r <= 'Z' {
			if i > 0 && !inBracket {
				result += "_"
			}
			result += strings.ToLower(string(r))
		} else {
			result += string(r)
		}
	}

	return strings.ToLower(result)
}

// generateFieldMessage generates a human-readable error message
func generateFieldMessage(field, tag, param string) string {
	msg := messages[tag]
	if msg != "" {
		msg = strings.ReplaceAll(msg, "{field}", field)
		msg = strings.ReplaceAll(msg, "{param}", param)
		return msg
	}
	return fmt.Sprintf("%s validation failed for tag '%s'", field, tag)
}
