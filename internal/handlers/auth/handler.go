package auth

import (
	"net/http"
	"oil/infras/otel"
	"oil/internal/domains/auth/model/dto"
	"oil/internal/domains/auth/service"
	"oil/shared/constant"
	"oil/shared/cookie"
	"oil/shared/failure"
	"oil/shared/validator"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service service.Auth
	otel    otel.Otel
}

func New(service service.Auth, otel otel.Otel) Handler {
	return Handler{
		service: service,
		otel:    otel,
	}
}

func (handler *Handler) Router(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", handler.Register)
		r.Post("/login", handler.Login)
		r.Post("/refresh-token", handler.RefreshToken)
	})
}

// Register handles user registration
// @Summary Register a new user
// @Description Register a new user with the provided details.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Register Request"
// @Success 201 {object} response.Message "User registered successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/auth/register [post]
func (handler *Handler) Register(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".Register")
	defer scope.End()

	req := dto.RegisterRequest{}

	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(w, err)

		return
	}

	if err := handler.service.Register(ctx, req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to create todo")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("User registered successfully")

	response.WithMessage(w, http.StatusCreated, "User registered successfully")
}

// Login handles user login
// @Summary Login a user
// @Description Login a user with the provided credentials. If 'remember' is true, the refresh token will be set as an HTTP-only cookie and excluded from the response body for security. If 'remember' is false, both tokens are returned in the response body.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login Request"
// @Success 200 {object} dto.LoginResponse "User logged in successfully. Response contains access_token. If remember=false, refresh_token is also included in response. If remember=true, refresh_token is only set as HTTP-only cookie."
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/auth/login [post]
func (handler *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".Login")
	defer scope.End()

	req := dto.LoginRequest{}

	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(w, err)

		return
	}

	res, err := handler.service.Login(ctx, req)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to login user")

		response.WithError(w, err)

		return
	}

	// If remember is true, set refresh token as HTTP-only cookie
	// and exclude it from JSON response for security
	if req.Remember {
		cookie.SetRefreshToken(w, res.RefreshToken)
		res.RefreshToken = constant.Empty // Clear from response body
	}

	scope.AddEvent("User logged in successfully")

	response.WithJSON(w, http.StatusOK, res)
}

// RefreshToken handles token refresh
// @Summary Refresh user token
// @Description Refresh user token using the refresh token from request body or HTTP-only cookie. If using cookie-based authentication, the new refresh token will only be set in the cookie and excluded from the response body for security.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest false "Refresh Token Request (optional if using cookie)"
// @Success 200 {object} dto.RefreshTokenResponse "Token refreshed successfully. Response contains access_token. If using cookie, new refresh_token is only in cookie. If using request body, new refresh_token is in response body."
// @Failure 400 {object} response.Error
// @Failure 422 {object} response.ValidationErrors
// @Failure 500 {object} response.Error
// @Router /v1/auth/refresh-token [post]
func (handler *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".RefreshTokenCookieName")
	defer scope.End()

	req := dto.RefreshTokenRequest{}

	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(w, err)

		return
	}

	// If no refresh token in body, try to get from cookie
	if req.RefreshToken == constant.Empty {
		if token, err := cookie.GetRefreshToken(r); err == nil {
			req.RefreshToken = token
		}
	}

	// Validate that we have a refresh token
	if req.RefreshToken == constant.Empty {
		err := &failure.ValidationError{
			Code: http.StatusUnprocessableEntity,
			Fields: []failure.ValidationFieldError{
				{
					Field:   "refresh_token",
					Message: "validation.required",
					Key:     "validation.required.refresh_token",
				},
			},
		}
		scope.TraceError(err)
		log.Error().Err(err).Msg("refresh token is required")

		response.WithError(w, err)

		return
	}

	res, err := handler.service.RefreshToken(ctx, req)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to refresh token")

		response.WithError(w, err)

		return
	}

	// If refresh token came from cookie, update the cookie with new refresh token
	// and exclude it from JSON response for security
	usingCookie := cookie.HasRefreshToken(r)
	if usingCookie {
		cookie.SetRefreshToken(w, res.RefreshToken)
		res.RefreshToken = constant.Empty // Clear from response body
	}

	scope.AddEvent("Token refreshed successfully")

	response.WithJSON(w, http.StatusOK, res)
}
