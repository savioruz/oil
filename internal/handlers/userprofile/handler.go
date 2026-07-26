package userprofile

import (
	"net/http"
	"oil/infras/otel"
	"oil/internal/domains/userprofile/model/dto"
	"oil/internal/domains/userprofile/service"
	"oil/shared"
	"oil/shared/constant"
	"oil/shared/failure"
	"oil/shared/validator"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service service.Userprofile
	otel    otel.Otel
}

func New(service service.Userprofile, otel otel.Otel) Handler {
	return Handler{
		service: service,
		otel:    otel,
	}
}

func (handler *Handler) Router(router chi.Router) {
	router.Route("/users", func(routerGroup chi.Router) {
		routerGroup.Get("/", handler.GetMe)
		routerGroup.Patch("/", handler.UpdateMe)
		routerGroup.Post("/presigned-url", handler.GeneratePresignedURL)
	})
}

// GetMe handles the retrieval of the authenticated user's profile.
// @Summary Get authenticated user's profile
// @Description Retrieve the profile information of the authenticated user.
// @Tags User - Profile
// @Produce json
// @Success 200 {object} dto.UserprofileResponse "User profile retrieved successfully"
// @Failure 401 {object} response.Error "Unauthorized"
// @Failure 500 {object} response.Error "Internal Server Error"
// @Router /api/users [get]
// @Security BearerAuth
func (handler *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GetMe")
	defer scope.End()

	userID := shared.GetUserID(ctx)
	if userID == "" {
		response.WithError(w, failure.Unauthorized("unauthorized"))

		return
	}

	profile, err := handler.service.Get(ctx, userID)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get userprofile")

		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, profile)
}

// UpdateMe handles the update of the authenticated user's profile.
// @Summary Update authenticated user's profile
// @Description Update the profile information of the authenticated user.
// @Tags User - Profile
// @Accept json
// @Produce json
// @Param request body dto.UpdateUserprofileRequest true "Update Userprofile Request"
// @Success 200 {object} response.Message "User profile updated successfully"
// @Failure 400 {object} response.Error "Bad Request"
// @Failure 401 {object} response.Error "Unauthorized"
// @Failure 500 {object} response.Error "Internal Server Error"
// @Router /api/users [patch]
// @Security BearerAuth
func (handler *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".UpdateMe")
	defer scope.End()

	userID := shared.GetUserID(ctx)
	if userID == "" {
		response.WithError(w, failure.Unauthorized("unauthorized"))

		return
	}

	req := dto.UpdateUserprofileRequest{}
	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(w, err)

		return
	}

	if err := handler.service.Update(ctx, req, userID); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to update userprofile")

		response.WithError(w, err)

		return
	}

	response.WithMessage(w, http.StatusOK, "Userprofile updated successfully")
}

func (handler *Handler) GeneratePresignedURL(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GeneratePresignedURL")
	defer scope.End()

	userID := shared.GetUserID(ctx)
	if userID == "" {
		response.WithError(w, failure.Unauthorized("unauthorized"))

		return
	}

	req := dto.GeneratePresignedURLRequest{}
	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(w, err)

		return
	}

	res, err := handler.service.GeneratePresignedURL(ctx, req, userID)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to generate presigned URL")

		response.WithError(w, err)

		return
	}

	response.WithJSON(w, http.StatusOK, res)
}
