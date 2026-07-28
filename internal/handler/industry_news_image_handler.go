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
