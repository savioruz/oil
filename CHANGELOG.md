# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Enhanced Authentication Security**: Refresh tokens now support HTTP-only cookie storage
  - **Cookie-based authentication** (when `remember=true`):
    - Login returns only `access_token` in response body
    - Refresh token is set as HTTP-only cookie (not in response body)
    - RefreshToken endpoint returns only `access_token` in response body
    - New refresh token is automatically updated in cookie
    - JavaScript cannot access refresh token (XSS protection)
  - **Traditional authentication** (when `remember=false`):
    - Login returns both `access_token` and `refresh_token` in response body
    - RefreshToken endpoint returns both tokens in response body
    - Client manages token storage
  - Cookie-based refresh tokens are automatically rotated on refresh
  - Cookies expiration matches JWT refresh token lifetime (configurable via `JWT_REFRESH_EXPIRE_MIN`, defaults to 7 days)
  - Cookies use `SameSite=Strict` for CSRF protection
  - **Frontend Impact**:
    - When `remember=true`: Store only `access_token` in memory, browser auto-sends refresh token via cookie
    - When `remember=false`: Store both tokens in memory/localStorage (traditional approach)
    - Refresh token endpoint accepts empty body when using cookies

- **Array-Based Validation Error Response**: Validation errors now return ALL field errors at once
  - Validation errors (422) return an `errors` array containing all validation failures
  - Each error contains `field` (snake_case field name, e.g., `"title"`, `"user_email"`, `"images[0]"`) and `message` (error key)
  - Example: `{"errors": [{"field": "title", "message": "validation.required"}, {"field": "user_email", "message": "validation.email"}]}`
  - Field names use JSON tag names and are automatically converted to snake_case
  - Array indices are preserved in field names (e.g., `"images[0]"`)
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
- **Breaking Change**: Cookie-based authentication now excludes refresh token from response body
  - When using cookie-based auth (`remember=true`), refresh token is NO LONGER returned in JSON response
  - Old behavior: `{"access_token": "...", "refresh_token": "..."}` + cookie set
  - New behavior: `{"access_token": "..."}` only + refresh token in HTTP-only cookie
  - This enhances security by preventing JavaScript access to refresh tokens
  - Traditional auth (`remember=false`) unchanged: both tokens still in response body
  - **Frontend Impact**:
    - Cookie-based: Only read `access_token` from response, refresh token is in cookie
    - Traditional: Read both `access_token` and `refresh_token` from response as before

- **Cookie MaxAge Configuration**: Cookie expiration now uses `JWT_REFRESH_EXPIRE_MIN` from config
  - Cookie `MaxAge` dynamically calculated from JWT refresh token expiration setting
  - Cookie lifetime automatically syncs with JWT configuration (defaults to 7 days/10080 minutes)
  - Single source of truth for refresh token expiration across tokens and cookies
  - Easily adjustable via `JWT_REFRESH_EXPIRE_MIN` environment variable

- **Breaking Change**: `remember` field in LoginRequest is now optional (defaults to `false`)
  - Old: `remember` was required in login requests
  - New: `remember` is optional, if omitted defaults to `false` (no cookie set)
  
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

### Fixed
- **Authentication HTTP Status Codes**: Fixed non-standard HTTP status codes in authentication endpoints
  - Invalid credentials (wrong email/password) now return **401 Unauthorized** instead of 400 Bad Request
  - Deactivated user accounts now return **403 Forbidden** instead of 400 Bad Request  
  - Email already registered now returns **409 Conflict** instead of 400 Bad Request
  - Invalid refresh token now uses proper error key `auth.token_invalid`
  - Wrong password in change password now returns **401 Unauthorized** instead of 400 Bad Request
  - Added new error keys: `auth.email_already_exists`, `auth.account_deactivated`
  - **Impact**: Frontend error handling should check for 401/403/409 status codes for proper user feedback

- **Validation Error Field Names**: Fixed issue where validation error `field` property was returning empty strings
  - Registered JSON tag name function to use `json` tag names instead of struct field names
  - Fixed `buildFieldPath` to properly extract field name from validator namespace
  - Field names now correctly show as `"title"`, `"user_email"`, `"images[0]"` instead of empty strings
  - Multi-word fields are automatically converted to snake_case (e.g., `UserEmail` → `user_email`)

### Security
- **Enhanced Refresh Token Security**: When using cookie-based authentication, refresh tokens are never exposed to JavaScript
  - Refresh tokens are only set as HTTP-only cookies (JavaScript cannot read them)
  - Refresh tokens are excluded from JSON response body when using cookies
  - This prevents XSS attacks from stealing refresh tokens
  - Only the access token is accessible to frontend JavaScript
- Improved security by never exposing internal error details in API responses
- All error responses now use standardized keys that don't leak implementation details

### Documentation
- Updated comprehensive `docs/Error.md` with:
  - Validation error response format with actual field and message examples
  - Complete validation message keys table showing array-based response format
  - Field name conventions (snake_case, array indices preserved)
  - Frontend integration examples showing field-level error handling with translation
  - Enhanced i18n implementation guide for multiple languages
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
