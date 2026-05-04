package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
	"github.com/healthcare-market-research/backend/pkg/validation"
)

type ReportImageHandler struct {
	service service.ReportImageService
}

func NewReportImageHandler(service service.ReportImageService) *ReportImageHandler {
	return &ReportImageHandler{service: service}
}

// UpdateImageMetadataRequest represents the request body for updating image metadata
type UpdateImageMetadataRequest struct {
	Title    *string `json:"title,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// UploadImage godoc
// @Summary Upload report image
// @Description Upload an image (chart, graph, diagram) for a report. Admin/editor only.
// @Tags Report Images
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param reportId path int true "Report ID"
// @Param image formData file true "Image file (max 10MB, allowed types: JPEG, PNG, WebP, GIF)"
// @Param title formData string false "Image title (optional, max 255 chars)"
// @Success 201 {object} response.Response{data=report.ReportImage} "Image uploaded successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports/{reportId}/images [post]
func (h *ReportImageHandler) UploadImage(c *fiber.Ctx) error {
	// Parse report ID
	reportID, err := strconv.ParseUint(c.Params("reportId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID")
	}

	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	// Get file from form
	file, err := c.FormFile("image")
	if err != nil {
		return response.BadRequest(c, "No image file provided")
	}

	// Validate image file
	if err := validation.ValidateImageFile(file); err != nil {
		return response.BadRequest(c, err.Error())
	}

	// Get optional title from form
	title := strings.TrimSpace(c.FormValue("title"))

	// Validate title length if provided
	if title != "" && (len(title) < 2 || len(title) > 255) {
		return response.BadRequest(c, "Title must be between 2 and 255 characters")
	}

	// Upload image
	image, err := h.service.UploadImage(uint(reportID), file, title, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return response.NotFound(c, "Report not found")
		}
		return response.InternalError(c, "Failed to upload image: "+err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(response.Response{
		Success: true,
		Data:    image,
	})
}

// ListImages godoc
// @Summary List report images
// @Description Get all images for a report with optional active filter. Admin/editor only.
// @Tags Report Images
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param reportId path int true "Report ID"
// @Param active query bool false "Filter by active status (true = only active images)"
// @Success 200 {object} response.Response{data=[]report.ReportImage} "List of images"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid report ID"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports/{reportId}/images [get]
func (h *ReportImageHandler) ListImages(c *fiber.Ctx) error {
	// Parse report ID
	reportID, err := strconv.ParseUint(c.Params("reportId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID")
	}

	// Parse active filter
	activeOnly := c.Query("active") == "true"

	// Get images
	images, err := h.service.GetImagesByReport(uint(reportID), activeOnly)
	if err != nil {
		return response.InternalError(c, "Failed to fetch images")
	}

	return response.Success(c, images)
}

// GetByID godoc
// @Summary Get single report image
// @Description Get a single report image by ID. Admin/editor only.
// @Tags Report Images
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param imageId path int true "Image ID"
// @Success 200 {object} response.Response{data=report.ReportImage} "Image details"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid image ID"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Image not found"
// @Router /api/v1/reports/images/{imageId} [get]
func (h *ReportImageHandler) GetByID(c *fiber.Ctx) error {
	// Parse image ID
	imageID, err := strconv.ParseUint(c.Params("imageId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid image ID")
	}

	// Get image
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
// @Summary Update report image metadata
// @Description Update image title and/or is_active status. Supports partial updates. Admin/editor only.
// @Tags Report Images
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param imageId path int true "Image ID"
// @Param metadata body UpdateImageMetadataRequest true "Metadata to update (all fields are optional for partial updates)"
// @Success 200 {object} response.Response{data=report.ReportImage} "Updated image"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Image not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports/images/{imageId} [patch]
func (h *ReportImageHandler) UpdateMetadata(c *fiber.Ctx) error {
	// Parse image ID
	imageID, err := strconv.ParseUint(c.Params("imageId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid image ID")
	}

	// Parse request body
	var req UpdateImageMetadataRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	// Validate title if provided
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title != "" && (len(title) < 2 || len(title) > 255) {
			return response.BadRequest(c, "Title must be between 2 and 255 characters")
		}
		req.Title = &title
	}

	// Update metadata
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
// @Summary Delete report image
// @Description Soft delete an image (sets is_active=false). Admin/editor only.
// @Tags Report Images
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
// @Router /api/v1/reports/images/{imageId} [delete]
func (h *ReportImageHandler) DeleteImage(c *fiber.Ctx) error {
	// Parse image ID
	imageID, err := strconv.ParseUint(c.Params("imageId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid image ID")
	}

	// Delete image (soft delete)
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
