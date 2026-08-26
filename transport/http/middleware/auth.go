package middleware

import (
	"context"
	"net/http"
	"oil/config"
	"oil/infras/jwt"
	"oil/infras/otel"
	"oil/internal/modules/userprofile/service"
	"oil/permissions"
	"oil/shared"
	"oil/shared/constant"
	"oil/shared/errkey"
	"oil/shared/failure"
	"oil/transport/http/response"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/rs/zerolog/log"
)

// SkipAuthKey is the context key to skip authentication for a route.
type SkipAuthKey string

// PermissionsKey is the context key for storing user permissions.
type PermissionsKey string

// Auth defines the interface for authentication middleware
type Auth interface {
	Auth(http.Handler) http.Handler
	APIKey(http.Handler) http.Handler
}

// Role defines the interface for role-based access control middleware
type Role interface {
	RBAC(http.Handler) http.Handler
}

// AuthRole combines all middleware interfaces
type AuthRole interface {
	Auth
	Role
}

// authRoleImpl implements the AuthRole interface
type authRoleImpl struct {
	jwtService         jwt.JWT
	userprofileService service.Userprofile
	otel               otel.Otel
	permission         *permissions.PermissionData
	cfg                *config.Config
}

// NewAuthRoleMiddleware creates a new middleware instance
func NewAuthRoleMiddleware(jwtService jwt.JWT, userprofileService service.Userprofile, otel otel.Otel, permissions *permissions.PermissionData, cfg *config.Config) AuthRole {
	return &authRoleImpl{
		jwtService:         jwtService,
		userprofileService: userprofileService,
		otel:               otel,
		permission:         permissions,
		cfg:                cfg,
	}
}

// Auth validates JWT tokens
// Requires valid authentication for all requests
func (m *authRoleImpl) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		_, scope := m.otel.NewScope(ctx, constant.OtelHandlerScopeName, "auth.middleware")

		skip, _ := request.Context().Value(SkipAuthKey("skip")).(bool)

		if skip {
			scope.End()
			next.ServeHTTP(writer, request)

			return
		}

		// Check if this endpoint should skip authentication based on permissions config
		if m.permission != nil {
			rctx := chi.RouteContext(ctx)
			method := request.Method
			path := rctx.Routes.Find(chi.NewRouteContext(), method, request.URL.Path)
			permission := m.permission.FindPermissions(path, method)

			if permission.Skip {
				scope.End()
				next.ServeHTTP(writer, request)

				return
			}
		}

		rctx := chi.RouteContext(ctx)
		method := request.Method
		path := rctx.Routes.Find(chi.NewRouteContext(), method, request.URL.Path)

		scope.SetAttributes(map[string]any{
			"middleware.type": "auth",
			"http.path":       path,
			"http.method":     method,
		})

		authHeader := request.Header.Get(constant.RequestHeaderAuthorization)
		if authHeader == "" {
			err := failure.Unauthorized("Missing authorization header")
			response.WithError(writer, err)

			scope.TraceError(err)
			scope.End()

			return
		}

		tokenString, err := jwt.ExtractTokenFromHeader(authHeader)
		if err != nil {
			err := failure.Unauthorized("Invalid authorization header format")
			response.WithError(writer, err)

			scope.TraceError(err)
			scope.End()

			return
		}

		claims, err := m.jwtService.ValidateToken(ctx, tokenString)
		if err != nil {
			errKey := errkey.ErrTokenInvalid
			message := "Token validation failed"

			errStr := err.Error()
			switch {
			case strings.Contains(errStr, "token_expired"):
				errKey = errkey.ErrTokenExpired
				message = "Token has expired"
			case strings.Contains(errStr, "invalid_token"):
				errKey = errkey.ErrTokenInvalid
				message = "Invalid token"
			case strings.Contains(errStr, "invalid_claim"):
				errKey = errkey.ErrInvalidClaim
				message = "Invalid token claims"
			case strings.Contains(errStr, "token_parse_failed"):
				errKey = errkey.ErrTokenInvalid
				message = "Token parse failed"
			case strings.Contains(errStr, "token_missing_kid"):
				errKey = errkey.ErrTokenInvalid
				message = "Token missing key ID"
			case strings.Contains(errStr, "token_invalid_claims"):
				errKey = errkey.ErrInvalidClaim
				message = "Token invalid claims"
			}

			err := failure.UnauthorizedWithKey(errKey, message)
			response.WithError(writer, err)

			scope.TraceError(err)
			scope.End()

			return
		}

		// Validate that required claims are not empty
		if claims.Subject == "" {
			log.Error().Msg("JWT claims: Subject is empty")

			response.WithError(writer, failure.Unauthorized("Invalid token claims"))
		}

		if claims.Email == "" {
			log.Error().Msg("JWT claims: Email is empty")

			response.WithError(writer, failure.Unauthorized("Invalid token claims"))
		}

		profile, err := m.userprofileService.GetOrCreateByAuthUserID(ctx, claims.Subject, claims.Email, claims.Role)
		if err != nil {
			log.Error().Err(err).Msg("failed to get or create userprofile")

			response.WithError(writer, failure.InternalError(err))
		}

		ctx = context.WithValue(ctx, constant.ContextKeyUserID, profile.ID)
		ctx = context.WithValue(ctx, constant.ContextKeyUserEmail, claims.Email)
		ctx = context.WithValue(ctx, constant.ContextKeyUserRole, profile.Role)

		scope.End()

		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// RBAC checks if user has required role
// Requires prior authentication via Auth middleware
func (m *authRoleImpl) RBAC(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		_, scope := m.otel.NewScope(ctx, constant.OtelHandlerScopeName, "rbac.middleware")

		skip, _ := request.Context().Value(SkipAuthKey("skip")).(bool)
		if skip {
			scope.End()
			next.ServeHTTP(writer, request)

			return
		}

		if m.permission == nil {
			scope.End()
			response.WithError(writer, failure.ForbiddenError)

			return
		}

		if m.permission.Skip {
			scope.End()
			next.ServeHTTP(writer, request)

			return
		}

		rctx := chi.RouteContext(request.Context())
		path := rctx.Routes.Find(chi.NewRouteContext(), request.Method, request.URL.Path)
		permission := m.permission.FindPermissions(path, request.Method)

		if permission.Skip {
			scope.End()
			next.ServeHTTP(writer, request)

			return
		}

		// Get user role from context
		userRole := shared.GetUserRole(ctx)

		// Check if user role is allowed (permissions field now contains roles)
		if len(permission.Permissions) > 0 {
			if !slices.Contains(permission.Permissions, userRole) {
				err := failure.ForbiddenError
				scope.TraceError(err)
				scope.SetAttributes(map[string]any{
					"user_role":     userRole,
					"allowed_roles": permission.Permissions,
					"reason":        "role_not_allowed",
				})
				scope.End()
				response.WithError(writer, err)

				return
			}
		}

		scope.End()
		next.ServeHTTP(writer, request)
	})
}

// APIKey for internal service-to-service authentication using API key
func (m *authRoleImpl) APIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		_, scope := m.otel.NewScope(ctx, constant.OtelHandlerScopeName, "api_key.middleware")

		ctx = context.WithValue(ctx, SkipAuthKey("skip"), false)
		apiKey := request.Header.Get(constant.RequestHeaderAPIKey)

		if apiKey == "" {
			scope.SetAttribute("http.source", "client")
			scope.End()
			next.ServeHTTP(writer, request.WithContext(ctx))

			return
		}

		scope.SetAttribute("http.source", "internal")

		if apiKey != m.cfg.App.APIKey {
			err := failure.ForbiddenError

			response.WithError(writer, failure.ForbiddenError)

			scope.TraceError(err)
			scope.End()

			return
		}

		ctx = context.WithValue(ctx, SkipAuthKey("skip"), true)

		scope.End()
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}
