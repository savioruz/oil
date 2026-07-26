package gallery

import (
	"mime/multipart"
	"net/http"
	"oil/infras/otel"
	"oil/infras/s3"
	"oil/internal/domains/gallery/model"
	"oil/internal/domains/gallery/model/dto"
	"oil/internal/domains/gallery/service"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/validator"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service service.Gallery
	s3      s3.S3
	otel    otel.Otel
}

func New(service service.Gallery, s3 s3.S3, otel otel.Otel) Handler {
	return Handler{
		service: service,
		s3:      s3,
		otel:    otel,
	}
}

func (handler *Handler) Router(router chi.Router) {
	router.Route("/galleries", func(routerGroup chi.Router) {
		routerGroup.Post("/", handler.CreateGallery)
		routerGroup.Get("/", handler.GetGalleries)
		routerGroup.Get("/{id}", handler.GetGalleryByID)
		routerGroup.Patch("/{id}", handler.UpdateGallery)
		routerGroup.Delete("/{id}", handler.DeleteGallery)
		routerGroup.Post("/upload", handler.UploadImage)
		routerGroup.Delete("/images", handler.DeleteImages)
	})
}

// CreateGallery handles the creation of a new gallery.
// @Summary Create a new gallery
// @Description Create a new gallery with the provided details.
// @Tags Gallery
// @Accept json
// @Produce json
// @Param request body dto.CreateGalleryRequest true "Create Gallery Request"
// @Success 201 {object} response.Message "Gallery created successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /api/galleries [post]
// @Security BearerAuth
func (handler *Handler) CreateGallery(writer http.ResponseWriter, request *http.Request) {
	ctx, scope := handler.otel.NewScope(request.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".CreateGallery")
	defer scope.End()

	req := dto.CreateGalleryRequest{}

	if err := validator.Validate(request.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(writer, err)

		return
	}

	if err := handler.service.Create(ctx, req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to create gallery")

		response.WithError(writer, err)

		return
	}

	user := shared.GetUserID(ctx)
	scope.AddEvent("Gallery created successfully by user " + user)

	response.WithMessage(writer, http.StatusCreated, "Gallery created successfully")
}

// GetGalleries retrieves all galleries based on query parameters.
// @Summary Get all galleries
// @Description Retrieve all galleries with optional filtering and pagination.
// @Tags Gallery
// @Accept json
// @Produce json
// @Param title query string false "Filter by title"
// @Param description query string false "Filter by description"
// @Success 200 {object} response.Data[dto.GetGalleriesResponse] "List of galleries"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /api/galleries [get]
func (handler *Handler) GetGalleries(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GetGalleries")
	defer scope.End()

	queryParams := gDto.QueryParams{}
	queryParams.FromRequest(r, true)

	title := r.URL.Query().Get(model.FieldTitle)
	description := r.URL.Query().Get(model.FieldDescription)

	const maxFilters = 3

	filterGroup := gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters:  make([]any, 0, maxFilters),
	}

	filterGroup.Filters = append(filterGroup.Filters, shared.UserFilter(ctx, constant.FieldCreatedBy, model.TableName))

	if title != "" {
		filterGroup.Filters = append(filterGroup.Filters, gDto.Filter{
			Field:    model.FieldTitle,
			Operator: gDto.FilterOperatorLike,
			Value:    title,
			Table:    model.TableName,
		})
	}

	if description != "" {
		filterGroup.Filters = append(filterGroup.Filters, gDto.Filter{
			Field:    model.FieldDescription,
			Operator: gDto.FilterOperatorLike,
			Value:    description,
			Table:    model.TableName,
		})
	}

	galleries, err := handler.service.GetAll(ctx, queryParams, filterGroup)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get galleries")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("Galleries retrieved successfully")

	response.WithJSON(w, http.StatusOK, galleries)
}

// GetGalleryByID retrieves a gallery by its ID.
// @Summary Get a gallery by ID
// @Description Retrieve a gallery by its unique identifier.
// @Tags Gallery
// @Accept json
// @Produce json
// @Param id path string true "Gallery ID"
// @Success 200 {object} response.Data[dto.GalleryResponse] "Gallery details"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /api/galleries/{id} [get]
func (handler *Handler) GetGalleryByID(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GetGalleryByID")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	gallery, err := handler.service.Get(ctx, id)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get gallery by ID")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("Gallery retrieved successfully")

	response.WithJSON(w, http.StatusOK, gallery)
}

// UpdateGallery updates an existing gallery by its ID.
// @Summary Update a gallery by ID
// @Description Update the details of an existing gallery.
// @Tags Gallery
// @Accept json
// @Produce json
// @Param id path string true "Gallery ID"
// @Param request body dto.UpdateGalleryRequest true "Update Gallery Request"
// @Success 200 {object} response.Message "Gallery updated successfully"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /api/galleries/{id} [patch]
// @Security BearerAuth
func (handler *Handler) UpdateGallery(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".UpdateGallery")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	req := dto.UpdateGalleryRequest{}
	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(w, err)

		return
	}

	if err := handler.service.Update(ctx, req, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to update gallery")

		response.WithError(w, err)

		return
	}

	user := shared.GetUserID(ctx)
	scope.AddEvent("Gallery updated successfully by user " + user)

	response.WithMessage(w, http.StatusOK, "Gallery updated successfully")
}

// DeleteGallery deletes a gallery by its ID.
// @Summary Delete a gallery by ID
// @Description Delete a gallery using its unique identifier.
// @Tags Gallery
// @Accept json
// @Produce json
// @Param id path string true "Gallery ID"
// @Success 200 {object} response.Message "Gallery deleted successfully"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /api/galleries/{id} [delete]
// @Security BearerAuth
func (handler *Handler) DeleteGallery(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".DeleteGallery")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	if err := handler.service.Delete(ctx, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to delete gallery")

		response.WithError(w, err)

		return
	}

	user := shared.GetUserID(ctx)
	scope.AddEvent("Gallery deleted successfully by user " + user)

	response.WithMessage(w, http.StatusOK, "Gallery deleted successfully")
}

// UploadImage handles image upload to S3.
// @Summary Upload an image to S3
// @Description Upload an image file to S3 and return the URL.
// @Tags Gallery
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Image file to upload"
// @Success 200 {object} dto.UploadImageResponse "Image uploaded successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /api/galleries/upload [post]
// @Security BearerAuth
func (handler *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".UploadImage")
	defer scope.End()

	if err := r.ParseMultipartForm(constant.RequestMaxMemory); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to parse multipart form")

		response.WithError(w, err)

		return
	}

	file, fileHeader, err := r.FormFile(constant.FormFile)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get file from form")

		response.WithError(w, err)

		return
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			log.Error().Err(err).Msg("failed to close file")
		}
	}(file)

	req := dto.UploadImageRequest{
		Image:     fileHeader,
		ImageFile: file,
	}

	res, err := handler.service.UploadImage(ctx, req)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to upload file")

		response.WithError(w, err)

		return
	}

	user := shared.GetUserID(ctx)
	scope.AddEvent("Image uploaded successfully by user " + user)

	response.WithJSON(w, http.StatusOK, res)
}

// DeleteImages handles deletion of multiple images from S3.
// @Summary Delete images from S3
// @Description Delete multiple images from S3 by providing their URLs.
// @Tags Gallery
// @Accept json
// @Produce json
// @Param request body dto.DeleteImagesRequest true "Delete Images Request"
// @Success 200 {object} response.Message "Images deleted successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /api/galleries/images [delete]
// @Security BearerAuth
func (handler *Handler) DeleteImages(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".DeleteImages")
	defer scope.End()

	req := dto.DeleteImagesRequest{}

	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(w, err)

		return
	}

	if err := handler.service.DeleteImagesFromS3(ctx, req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to delete images from S3")

		response.WithError(w, err)

		return
	}

	user := shared.GetUserID(ctx)
	scope.AddEvent("Images deleted successfully by user " + user)

	response.WithMessage(w, http.StatusOK, "Images deleted successfully")
}
