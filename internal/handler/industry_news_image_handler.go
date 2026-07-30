package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
	"github.com/healthcare-market-research/backend/pkg/validation"
)

type IndustryNewsImageHandler struct {
	service service.IndustryNewsImageService
}

func NewIndustryNewsImageHandler(service service.IndustryNewsImageService) *IndustryNewsImageHandler {
	return &IndustryNewsImageHandler{service: service}
}

// UploadImage godoc
// @Summary Upload industry news image
// @Description Upload an image for an industry news article (for use in the content editor). Admin/editor only.
// @Tags Industry News Images
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param newsId path int true "Industry news article ID"
// @Param image formData file true "Image file (max 10MB, allowed types: JPEG, PNG, WebP, GIF)"
// @Param title formData string false "Image title (optional, max 255 chars)"
// @Success 201 {object} response.Response{data=industry_news.IndustryNewsImage} "Image uploaded successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Industry news article not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/industry-news/{newsId}/images [post]
func (h *IndustryNewsImageHandler) UploadImage(c *fiber.Ctx) error {
	industryNewsID, err := strconv.ParseUint(c.Params("newsId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID")
	}

	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	file, err := c.FormFile("image")
	if err != nil {
		return response.BadRequest(c, "No image file provided")
	}

	if err := validation.ValidateImageFile(file); err != nil {
		return response.BadRequest(c, err.Error())
	}

	title := strings.TrimSpace(c.FormValue("title"))

	if title != "" && (len(title) < 2 || len(title) > 255) {
		return response.BadRequest(c, "Title must be between 2 and 255 characters")
	}

	image, err := h.service.UploadImage(uint(industryNewsID), file, title, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return response.NotFound(c, "Industry news article not found")
		}
		return response.InternalError(c, "Failed to upload image: "+err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(response.Response{
		Success: true,
		Data:    image,
	})
}

// ListImages godoc
// @Summary List industry news images
// @Description Get all images for an industry news article with optional active filter. Admin/editor only.
// @Tags Industry News Images
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param newsId path int true "Industry news article ID"
// @Param active query bool false "Filter by active status (true = only active images)"
// @Success 200 {object} response.Response{data=[]industry_news.IndustryNewsImage} "List of images"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid industry news ID"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/industry-news/{newsId}/images [get]
func (h *IndustryNewsImageHandler) ListImages(c *fiber.Ctx) error {
	industryNewsID, err := strconv.ParseUint(c.Params("newsId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID")
	}

	activeOnly := c.Query("active") == "true"

	images, err := h.service.GetImagesByIndustryNews(uint(industryNewsID), activeOnly)
	if err != nil {
		return response.InternalError(c, "Failed to fetch images")
	}

	return response.Success(c, images)
}

// GetByID godoc
// @Summary Get single industry news image
// @Description Get a single industry news image by ID. Admin/editor only.
// @Tags Industry News Images
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param imageId path int true "Image ID"
// @Success 200 {object} response.Response{data=industry_news.IndustryNewsImage} "Image details"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid image ID"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Image not found"
// @Router /api/v1/industry-news/images/{imageId} [get]
func (h *IndustryNewsImageHandler) GetByID(c *fiber.Ctx) error {
	imageID, err := strconv.ParseUint(c.Params("imageId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid image ID")
	}

	image, err := h.service.GetImageByID(uint(imageID))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return response.NotFound(c, "Image not found")
		}
		return response.InternalError(c, "Failed to fetch image")
	}

	return response.Success(c, image)
}

// UpdateMetadata godoc
// @Summary Update industry news image metadata
// @Description Update image title and/or is_active status. Supports partial updates. Admin/editor only.
// @Tags Industry News Images
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param imageId path int true "Image ID"
// @Param metadata body UpdateImageMetadataRequest true "Metadata to update (all fields are optional for partial updates)"
// @Success 200 {object} response.Response{data=industry_news.IndustryNewsImage} "Updated image"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Image not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/industry-news/images/{imageId} [patch]
func (h *IndustryNewsImageHandler) UpdateMetadata(c *fiber.Ctx) error {
	imageID, err := strconv.ParseUint(c.Params("imageId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid image ID")
	}

	var req UpdateImageMetadataRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title != "" && (len(title) < 2 || len(title) > 255) {
			return response.BadRequest(c, "Title must be between 2 and 255 characters")
		}
		req.Title = &title
	}

	image, err := h.service.UpdateImageMetadata(uint(imageID), req.Title, req.IsActive)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return response.NotFound(c, "Image not found")
		}
		return response.InternalError(c, "Failed to update image metadata: "+err.Error())
	}

	return response.Success(c, image)
}

// DeleteImage godoc
// @Summary Delete industry news image
// @Description Delete an industry news image. Admin/editor only.
// @Tags Industry News Images
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param imageId path int true "Image ID"
// @Success 200 {object} response.Response{data=map[string]string} "Image deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid image ID"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Image not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/industry-news/images/{imageId} [delete]
func (h *IndustryNewsImageHandler) DeleteImage(c *fiber.Ctx) error {
	imageID, err := strconv.ParseUint(c.Params("imageId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid image ID")
	}

	if err := h.service.DeleteImage(uint(imageID)); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return response.NotFound(c, "Image not found")
		}
		return response.InternalError(c, "Failed to delete image: "+err.Error())
	}

	return response.Success(c, fiber.Map{
		"message": "Image deleted successfully",
	})
}
