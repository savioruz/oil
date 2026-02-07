# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Field-Specific Validation Error Keys**: Enhanced validation error system to return field-specific error keys
  - Validation errors now return keys in format: `validation.{rule}.{field_name}`
  - Examples: `validation.required.title`, `validation.url.images`, `validation.min.title`
  - Added `ValidationError` type with field-level error details
  - Added `FormatFieldError()` and `ToSnakeCase()` helper functions in errkey package
  - Added validation tag to error key mapping
  - Enhanced validator package to build structured validation errors
  - Updated response package to return first field's error key for validation errors

- **Error Key System**: Implemented machine-readable error keys for all API responses
  - Added `shared/errkey` package with 51+ standardized error keys using dot notation
  - Enhanced `shared/failure` package with `*WithKey()` helper functions
  - Added intelligent `GetKey()` function that provides default keys based on HTTP status codes
  - Added comprehensive error documentation in `docs/Error.md`
  - Created complete test suite for error key functionality

### Changed
- **Breaking Change**: Validation errors now return field-specific error keys
  - Old format: `{"error": "validation.failed"}`
  - New format: `{"error": "validation.required.title"}` or `{"error": "validation.url.images"}`
  - When multiple validation errors occur, the first error's key is returned
  - This allows frontends to show field-specific error messages and highlight problematic fields

- **Breaking Change**: API error responses now return error keys instead of error messages
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
