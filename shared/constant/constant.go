package constant

import (
	"time"
)

const (
	ContextGuest = "guest"
)

// Context key types to avoid collisions
type contextKey string

const (
	ContextKeyUserID    contextKey = "user_id"
	ContextKeyUserEmail contextKey = "user_email"
	ContextKeyUserRole  contextKey = "user_role"
	ContextKeyTokenID   contextKey = "token_id"
)

const (
	RoleSuperAdmin = "superadmin"
	RoleAdmin      = "admin"
	RoleUser       = "user"
)

const (
	RequestParamPage    = "page"
	RequestParamLimit   = "limit"
	RequestParamSortBy  = "sort_by"
	RequestParamSortDir = "sort_dir"
)

const (
	RequestParamID   = "id"
	RequestMaxMemory = 1 << 20 // 1 MB
)

const (
	DefaultValuePage    = 1
	DefaultValueLimit   = 10
	DefaultValueSortBy  = "created_at"
	DefaultValueSortDir = "DESC"
)

const (
	FieldCreatedAt  = "created_at"
	FieldCreatedBy  = "created_by"
	FieldModifiedAt = "modified_at"
	FieldModifiedBy = "modified_by"
)

const (
	PqErrorCodeUniqueViolation = "23505"
	PqErrorCodeFkViolation     = "23503"
)

const (
	DateFormat = time.RFC3339
)

const (
	MinutesToSeconds = 60
)

const (
	OtelServiceScopeName    = "service"
	OtelRepositoryScopeName = "repository"
	OtelHandlerScopeName    = "handler"
	OtelEventScopeName      = "event"
	OtelExternalScopeName   = "external"

	OtelQueryAttributeKey = "query"
	OtelS3ScopeName       = "s3"
)

const (
	RequestHeaderAuthorization      = "Authorization"
	RequestHeaderUserAgent          = "User-Agent"
	RequestHeaderContentType        = "Content-Type"
	RequestHeaderRateLimit          = "X-RateLimit-Limit"
	RequestHeaderRateLimitRemaining = "X-RateLimit-Remaining"
	RequestHeaderRateLimitWindow    = "X-RateLimit-Window"
	RequestHeaderRequestID          = "X-Request-ID"
	RequestHeaderForwardedFor       = "X-Forwarded-For"
	RequestHeaderRealIP             = "X-Real-IP"
	RequestHeaderAPIKey             = "X-API-Key"
)

const (
	ContentTypeJSON              = "application/json"
	ContentTypeFormURLEncoded    = "application/x-www-form-urlencoded"
	ContentTypeMultipartFormData = "multipart/form-data"
	FormFile                     = "file"
)

const (
	ResponseErrorPrepareShutdown      = "SERVER PREPARING TO SHUT DOWN"
	ResponseErrorUnhealthy            = "SERVER UNHEALTHY"
	ResponseErrorRequestLimitExceeded = "REQUEST LIMIT EXCEEDED"
)

const (
	ServerEnvDevelopment = "development"
	ServerEnvProduction  = "production"
)

const (
	Asterix = "*"
	Empty   = ""
)
