package handler

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
	"github.com/healthcare-market-research/backend/pkg/validation"
)

type CategoryHandler struct {
	service service.CategoryService
}

func NewCategoryHandler(service service.CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

// GetAll godoc
// @Summary Get all categories
// @Description Get a paginated list of all active healthcare report categories
// @Tags Categories
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1, min: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} response.Response{data=[]category.Category,meta=response.Meta} "List of categories with pagination metadata"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/categories [get]
func (h *CategoryHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	categories, total, err := h.service.GetAll(page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to fetch categories")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response.SuccessWithMeta(c, categories, meta)
}

// GetBySlug godoc
// @Summary Get category by slug
// @Description Get a single healthcare report category by its unique slug
// @Tags Categories
// @Accept json
// @Produce json
// @Param slug path string true "Category slug"
// @Success 200 {object} response.Response{data=category.Category} "Category details"
// @Failure 400 {object} response.Response{error=string} "Bad request - slug is required"
// @Failure 404 {object} response.Response{error=string} "Category not found"
// @Router /api/v1/categories/{slug} [get]
func (h *CategoryHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.BadRequest(c, "Slug is required")
	}

	category, err := h.service.GetBySlug(slug)
	if err != nil {
		return response.NotFound(c, "Category not found")
	}

	return response.Success(c, category)
}

// UploadImage godoc
// @Summary Upload category image
// @Description Upload or replace the feature image for a category. Requires admin or editor role.
// @Tags Categories
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Category ID"
// @Param file formData file true "Image file (max 10MB, allowed types: JPEG, PNG, WebP)"
// @Success 200 {object} response.Response{data=category.Category} "Category with updated image URL"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID, missing file, or validation error"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Category not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/categories/{id}/image [post]
func (h *CategoryHandler) UploadImage(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid category ID")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "No image file provided")
	}

	if err := validation.ValidateImageFile(file); err != nil {
		return response.BadRequest(c, err.Error())
	}

	updatedCategory, err := h.service.UploadImage(uint(id), file)
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Category not found")
		}
		return response.InternalError(c, "Failed to upload image: "+err.Error())
	}

	return response.Success(c, updatedCategory)
}
