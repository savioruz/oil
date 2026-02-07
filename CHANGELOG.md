# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Array-Based Validation Error Response**: Validation errors now return ALL field errors at once
  - Validation errors (422) return an `errors` array containing all validation failures
  - Each error contains `field` (snake_case field name) and `message` (human-readable text)
  - Example: `{"errors": [{"field": "title", "message": "Title is required"}, {"field": "email", "message": "Email must be a valid email address"}]}`
  - Non-validation errors still return single error key: `{"error": "gallery.not_found"}`
  - This allows frontends to highlight all problematic fields in a single request/response cycle

- **Human-Readable Validation Messages**: Validation error messages now return error keys for frontend translation
  - Messages are error keys like `"validation.required"`, `"validation.email"`, `"validation.min"`
  - Frontend can translate these keys to any language and customize wording
  - Error keys are simpler and don't include field names (just the validation rule)
  - See `shared/validator/message.go` for complete list of validation error keys

- **Field-Specific Validation Error Keys**: Enhanced validation error system to return field-specific error keys
  - Validation errors now return keys in format: `validation.{rule}.{field_name}`
  - Examples: `validation.required.title`, `validation.url.images`, `validation.min.title`
  - Added `ValidationError` type with field-level error details
  - Added `FormatFieldError()` and `ToSnakeCase()` helper functions in errkey package
  - Added validation tag to error key mapping
  - Enhanced validator package to build structured validation errors

- **Error Key System**: Implemented machine-readable error keys for all API responses
  - Added `shared/errkey` package with 51+ standardized error keys using dot notation
  - Enhanced `shared/failure` package with `*WithKey()` helper functions
  - Added intelligent `GetKey()` function that provides default keys based on HTTP status codes
  - Added comprehensive error documentation in `docs/Error.md`
  - Created complete test suite for error key functionality

### Changed
- **Breaking Change**: Validation error message field now returns error keys instead of human-readable text
  - Old format: `{"errors": [{"field": "title", "message": "Title is required"}]}`
  - New format: `{"errors": [{"field": "title", "message": "validation.required"}]}`
  - The `message` field now contains error keys (e.g., `"validation.required"`, `"validation.email"`)
  - Frontend must translate error keys to localized messages
  - Error keys are simpler and language-agnostic (no field names in message)
  - **Frontend Impact**: 
    - Create translation map for validation error keys
    - Translate keys like `"validation.required"` → `"Title is required"` (en) or `"Título es obligatorio"` (es)
    - Field name is already available in the `field` property
  - See `docs/Error.md` for complete frontend translation examples

- **Breaking Change**: Validation error response format changed to array-based format
  - Old format: `{"error": "validation.required.title"}` (single field only)
  - New format: `{"errors": [{"field": "title", "message": "validation.required"}, {"field": "images", "message": "validation.required"}]}` (all fields)
  - Non-validation errors unchanged: `{"error": "gallery.not_found"}`
  - **Frontend Impact**: Frontends must handle two different response formats:
    - For 422 status: Check `response.data.errors` array
    - For other errors: Check `response.data.error` string
  - See `docs/Error.md` for complete frontend integration examples

- **Breaking Change**: API error responses now return error keys instead of error messages (for non-validation errors)
  - Old format: `{"error": "gallery not found"}`
  - New format: `{"error": "gallery.not_found"}`
  - All error responses now return machine-readable keys consistently
  - Frontend applications should map error keys to localized user-friendly messages

- Updated all service layer errors to use `*WithKey()` functions
- Updated `transport/http/response` package to always return error keys
- Simplified response logic by removing environment-dependent error formatting

### Security
- Improved security by never exposing internal error details in API responses
- All error responses now use standardized keys that don't leak implementation details

### Documentation
- Updated comprehensive `docs/Error.md` with:
  - Field-specific validation error key patterns and examples
  - Complete error key reference table with validation patterns
  - Frontend integration examples showing field-level error handling
  - Enhanced i18n implementation guide with pattern matching
  - API response examples for various error scenarios
  - Error handling best practices
  - Migration guide for existing clients

### Testing
- Added test for validation errors with field-specific keys
- Added `TestValidationErrorExamples` demonstrating various validation scenarios
- Added integration test file: `validation_integration_test.go`
- All 78+ tests passing with improved coverage
- Added `shared/errkey/errkey_test.go` for error key validation

## Migration Guide

If you're upgrading from a previous version, please note:

1. **Validation Error Format Changed**: Validation errors now return field-specific keys
   - Update frontend to handle keys like `validation.required.title` instead of `validation.failed`
   - Extract field name from error key to highlight the problematic form field
   - Use pattern matching for generic fallback messages
2. **API Response Format Changed**: Error responses now contain keys instead of messages
3. **Frontend Updates Required**: Update error handling to map keys to messages
4. **No Server Configuration Needed**: System works consistently across all environments

See `docs/Error.md` for detailed migration instructions and integration examples.
