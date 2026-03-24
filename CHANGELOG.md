# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - 2026-03-23

### Added

- **External Auth Service Integration**
  - JWT validation via JWKS from external auth service (`AUTH_SERVICE_URL`)
  - JWT signing: EdDSA/Ed25519 with OKP key type
  - Issuer validation: accepts `AUTH_SERVICE_URL` or `https://auth.svrz.xyz`
  - Audience validation: accepts `APP_NAME` or `oil-local`
  - One-time JWKS fetch at startup using `sync.Once`

- **User Profile Domain**
  - New domain: `internal/domains/userprofile/`
  - Lazy sync: First API call creates/links profile from JWT claims (`sub` → `auth_user_id`)
  - Endpoints: `GET /api/users`, `PATCH /api/users`, `POST /api/users/presigned-url`

- **S3 Presigned URL Support**
  - New method: `GetPresignedUploadURL` in `infras/s3/s3.go`
  - Used for avatar uploads (max 1MB)

- **Feature Flags with Unleash**
  - Added `infras/unleash` package with `FeatureFlag` interface
  - Fail-open behavior when Unleash is unreachable
  - Config via `UNLEASH_URL`, `UNLEASH_APP_NAME`, `UNLEASH_INSTANCE_ID`, `UNLEASH_SECRET`, `UNLEASH_ENVIRONMENT`

### Removed

- **Local Authentication**
  - Removed `internal/domains/auth/` (service, model, dto)
  - Removed auth handler: `internal/handlers/auth/handler.go`
  - Removed cookie-based token storage (now uses Bearer token only)
  - Removed: `email_verifications` table, `password_resets` table

- **JWT Configuration**
  - Removed: `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET`, `JWT_ACCESS_EXPIRE_MIN`, `JWT_REFRESH_EXPIRE_MIN`

- **Users Table**
  - Replaced with `user_profiles` table
  - Removed columns: `password`, `google_id`
  - Renamed domain: `internal/domains/user/` → `internal/domains/userprofile/`

### Changed

- **JWT-based Authentication**
  - Bearer token in Authorization header only
  - Token validation: EdDSA/Ed25519

- **Array-Based Validation Error Response**
  - Validation errors (422) return all field errors at once
  - Each error contains `field` and `message` (error key)
  - Example: `{"errors": [{"field": "title", "message": "validation.required"}]}`

- **Unified Structured Logging**
  - Both development and production use structured JSON logging
  - Enhanced OpenTelemetry spans with additional request/response attributes

## [Unreleased] - 2025-07-31

### Added

- Initial project setup
- Todo management with CRUD operations
- Gallery management with image uploads
- JWT-based authentication
- Role-based access control (RBAC)
- Rate limiting
- OpenAPI/Swagger documentation
