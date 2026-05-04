package handler

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/author"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
	"github.com/healthcare-market-research/backend/pkg/validation"
)

type AuthorHandler struct {
	service service.AuthorService
}

func NewAuthorHandler(service service.AuthorService) *AuthorHandler {
	return &AuthorHandler{service: service}
}

// GetAll godoc
// @Summary Get all authors
// @Description Get a paginated list of authors with optional search. Response includes linkedinUrl for each author if set.
// @Tags Authors
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1, min: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Param search query string false "Search in name, role, and bio"
// @Success 200 {object} response.Response{data=[]author.Author,meta=response.Meta} "List of authors with pagination metadata"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/authors [get]
func (h *AuthorHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	search := c.Query("search", "")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	authors, total, err := h.service.GetAll(page, limit, search)
	if err != nil {
		return response.InternalError(c, "Failed to fetch authors")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response.SuccessWithMeta(c, authors, meta)
}

// GetByID godoc
// @Summary Get single author
// @Description Get a single author by ID. Response includes linkedinUrl if set.
// @Tags Authors
// @Accept json
// @Produce json
// @Param id path int true "Author ID"
// @Success 200 {object} response.Response{data=author.Author} "Author details"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Author not found"
// @Router /api/v1/authors/{id} [get]
func (h *AuthorHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid author ID")
	}

	author, err := h.service.GetByID(uint(id))
	if err != nil {
		return response.NotFound(c, "Author not found")
	}

	return response.Success(c, author)
}

// Create godoc
// @Summary Create author
// @Description Create a new author. Requires admin or editor role.
// @Tags Authors
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param author body author.Author true "Author data (name is required with min 2 chars, role, bio, and linkedinUrl are optional. linkedinUrl must be a valid HTTPS URL)"
// @Success 201 {object} response.Response{data=author.Author} "Created author"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/authors [post]
func (h *AuthorHandler) Create(c *fiber.Ctx) error {
	var req author.Author
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	// Validate required fields
	if req.Name == "" {
		return response.BadRequest(c, "Name is required")
	}
	if len(req.Name) < 2 {
		return response.BadRequest(c, "Name must be at least 2 characters")
	}

	// Validate bio length if provided
	if len(req.Bio) > 1000 {
		return response.BadRequest(c, "Bio must not exceed 1000 characters")
	}

	// Validate LinkedIn URL if provided
	if req.LinkedinURL != "" {
		if err := validation.ValidateURL(req.LinkedinURL); err != nil {
			return response.BadRequest(c, "Invalid LinkedIn URL: "+err.Error())
		}
	}

	if err := h.service.Create(&req); err != nil {
		return response.InternalError(c, "Failed to create author")
	}

	return c.Status(fiber.StatusCreated).JSON(response.Response{
		Success: true,
		Data:    req,
	})
}

// Update godoc
// @Summary Update author
// @Description Update an existing author by ID. Supports partial updates. Requires admin or editor role.
// @Tags Authors
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Author ID"
// @Param author body author.Author true "Updated author data (all fields are optional for partial updates. linkedinUrl can be updated or cleared, must be a valid HTTPS URL if provided)"
// @Success 200 {object} response.Response{data=author.Author} "Updated author"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Author not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/authors/{id} [put]
func (h *AuthorHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid author ID")
	}

	// Get existing author to support partial updates
	existing, err := h.service.GetByID(uint(id))
	if err != nil {
		return response.NotFound(c, "Author not found")
	}

	var req author.Author
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	// Support partial updates - only update fields that are provided
	if req.Name != "" {
		if len(req.Name) < 2 {
			return response.BadRequest(c, "Name must be at least 2 characters")
		}
		existing.Name = req.Name
	}

	// Update role if provided (even if empty string to allow clearing)
	if c.Request().Body() != nil {
		bodyMap := make(map[string]interface{})
		if err := c.BodyParser(&bodyMap); err == nil {
			if _, ok := bodyMap["role"]; ok {
				existing.Role = req.Role
			}
			if _, ok := bodyMap["bio"]; ok {
				if len(req.Bio) > 1000 {
					return response.BadRequest(c, "Bio must not exceed 1000 characters")
				}
				existing.Bio = req.Bio
			}
			if _, ok := bodyMap["linkedinUrl"]; ok {
				if req.LinkedinURL != "" {
					if err := validation.ValidateURL(req.LinkedinURL); err != nil {
						return response.BadRequest(c, "Invalid LinkedIn URL: "+err.Error())
					}
				}
				existing.LinkedinURL = req.LinkedinURL
			}
		}
	}

	if err := h.service.Update(uint(id), existing); err != nil {
		return response.InternalError(c, "Failed to update author")
	}

	return response.Success(c, existing)
}

// Delete godoc
// @Summary Delete author
// @Description Delete an author by ID. Cannot delete if author is referenced in any reports. Requires admin role.
// @Tags Authors
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Author ID"
// @Success 200 {object} response.Response{data=map[string]string} "Author deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or author is referenced in reports"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin role"
// @Failure 404 {object} response.Response{error=string} "Author not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/authors/{id} [delete]
func (h *AuthorHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid author ID")
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Author not found")
		}
		if err.Error() == "cannot delete author: author is referenced in one or more reports" {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "Failed to delete author")
	}

	return response.Success(c, fiber.Map{
		"message": "Author deleted successfully",
	})
}

// UploadImage godoc
// @Summary Upload author image
// @Description Upload or replace profile image for an author. Requires admin or editor role.
// @Tags Authors
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Author ID"
// @Param image formData file true "Image file (max 10MB, allowed types: JPEG, PNG, WebP, GIF)"
// @Success 200 {object} response.Response{data=author.Author} "Author with updated image URL"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID, missing file, or validation error"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Author not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/authors/{id}/image [post]
func (h *AuthorHandler) UploadImage(c *fiber.Ctx) error {
	// Parse author ID
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid author ID")
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

	// Upload image
	updatedAuthor, err := h.service.UploadImage(uint(id), file)
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Author not found")
		}
		return response.InternalError(c, "Failed to upload image: "+err.Error())
	}

	return response.Success(c, updatedAuthor)
}

// DeleteImage godoc
// @Summary Delete author image
// @Description Delete the profile image of an author. Requires admin or editor role.
// @Tags Authors
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Author ID"
// @Success 200 {object} response.Response{data=map[string]string} "Image deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or author has no image"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Author not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/authors/{id}/image [delete]
func (h *AuthorHandler) DeleteImage(c *fiber.Ctx) error {
	// Parse author ID
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid author ID")
	}

	// Delete image
	if err := h.service.DeleteImage(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Author not found")
		}
		if err.Error() == "author has no image to delete" {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "Failed to delete image: "+err.Error())
	}

	return response.Success(c, fiber.Map{
		"message": "Image deleted successfully",
	})
}
