# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Error Key System**: Implemented machine-readable error keys for all API responses
  - Added `shared/errkey` package with 51+ standardized error keys using dot notation
  - Enhanced `shared/failure` package with `*WithKey()` helper functions
  - Added intelligent `GetKey()` function that provides default keys based on HTTP status codes
  - Added comprehensive error documentation in `docs/Error.md`
  - Created complete test suite for error key functionality

### Changed
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
- Added comprehensive `docs/Error.md` with:
  - Complete error key reference table
  - Frontend integration examples (TypeScript/React)
  - i18n implementation guide
  - Error handling best practices
  - Migration guide for existing clients

### Testing
- Added 10 new test cases for response package error key functionality
- Updated 61 existing handler tests to validate error key responses
- All 76 tests passing with improved coverage
- Added `shared/errkey/errkey_test.go` for error key validation

## Migration Guide

If you're upgrading from a previous version, please note:

1. **API Response Format Changed**: Error responses now contain keys instead of messages
2. **Frontend Updates Required**: Update error handling to map keys to messages
3. **No Server Configuration Needed**: System works consistently across all environments

See `docs/Error.md` for detailed migration instructions and integration examples.
