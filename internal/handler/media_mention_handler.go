package handler

import (
	"fmt"
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/mediamention"
	"github.com/healthcare-market-research/backend/internal/domain/report"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
	"github.com/healthcare-market-research/backend/pkg/validation"
)

// ReportLookup is the minimal capability MediaMentionHandler needs from
// ReportRepository: looking up a report by ID to validate a reportId
// reference before persisting it. Satisfied structurally by
// repository.ReportRepository — no adapter needed at the call site.
type ReportLookup interface {
	GetByID(id uint) (*report.Report, error)
}

type MediaMentionHandler struct {
	service      service.MediaMentionService
	reportLookup ReportLookup
}

func NewMediaMentionHandler(service service.MediaMentionService, reportLookup ReportLookup) *MediaMentionHandler {
	return &MediaMentionHandler{service: service, reportLookup: reportLookup}
}

// validateReportReference checks that reportID (if present) points at a
// published, non-deleted report, and that reportID/reportLinkText are
// either both set or both empty.
func (h *MediaMentionHandler) validateReportReference(reportID *uint, reportLinkText string) error {
	if reportID == nil && reportLinkText == "" {
		return nil
	}
	if reportID == nil || reportLinkText == "" {
		return fmt.Errorf("reportId and reportLinkText must be set together")
	}

	rep, err := h.reportLookup.GetByID(*reportID)
	if err != nil || rep.Status != "published" || rep.DeletedAt != nil {
		return fmt.Errorf("selected report not found or not published")
	}
	return nil
}

// GetAll godoc
// @Summary Get all media mentions
// @Description Get a paginated list of media mentions with optional search
// @Tags MediaMentions
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1, min: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Param search query string false "Search in title"
// @Success 200 {object} response.Response{data=[]mediamention.MediaMention,meta=response.Meta} "List of media mentions with pagination metadata"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/media-mentions [get]
func (h *MediaMentionHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	search := c.Query("search", "")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	mentions, total, err := h.service.GetAll(page, limit, search)
	if err != nil {
		return response.InternalError(c, "Failed to fetch media mentions")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response.SuccessWithMeta(c, mentions, meta)
}

// GetByID godoc
// @Summary Get single media mention
// @Tags MediaMentions
// @Accept json
// @Produce json
// @Param id path int true "Media Mention ID"
// @Success 200 {object} response.Response{data=mediamention.MediaMention}
// @Failure 400 {object} response.Response{error=string}
// @Failure 404 {object} response.Response{error=string}
// @Router /api/v1/media-mentions/{id} [get]
func (h *MediaMentionHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid media mention ID")
	}

	mention, err := h.service.GetByID(uint(id))
	if err != nil {
		return response.NotFound(c, "Media mention not found")
	}

	return response.Success(c, mention)
}

// Create godoc
// @Summary Create media mention
// @Description Create a new media mention. Requires admin or editor role.
// @Tags MediaMentions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param mediaMention body mediamention.MediaMention true "Media mention data (title is required, link must be a valid URL if provided)"
// @Success 201 {object} response.Response{data=mediamention.MediaMention}
// @Failure 400 {object} response.Response{error=string}
// @Failure 401 {object} response.Response{error=string}
// @Failure 403 {object} response.Response{error=string}
// @Failure 500 {object} response.Response{error=string}
// @Router /api/v1/media-mentions [post]
func (h *MediaMentionHandler) Create(c *fiber.Ctx) error {
	var req mediamention.MediaMention
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	if req.Title == "" {
		return response.BadRequest(c, "Title is required")
	}
	if len(req.Title) < 2 {
		return response.BadRequest(c, "Title must be at least 2 characters")
	}

	if req.Link != "" {
		if err := validation.ValidateURL(req.Link); err != nil {
			return response.BadRequest(c, "Invalid link URL: "+err.Error())
		}
	}

	if err := h.validateReportReference(req.ReportID, req.ReportLinkText); err != nil {
		return response.BadRequest(c, err.Error())
	}

	if err := h.service.Create(&req); err != nil {
		return response.InternalError(c, "Failed to create media mention")
	}

	return c.Status(fiber.StatusCreated).JSON(response.Response{
		Success: true,
		Data:    req,
	})
}

// Update godoc
// @Summary Update media mention
// @Description Update an existing media mention by ID. Supports partial updates. Requires admin or editor role.
// @Tags MediaMentions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Media Mention ID"
// @Param mediaMention body mediamention.MediaMention true "Updated media mention data"
// @Success 200 {object} response.Response{data=mediamention.MediaMention}
// @Failure 400 {object} response.Response{error=string}
// @Failure 401 {object} response.Response{error=string}
// @Failure 403 {object} response.Response{error=string}
// @Failure 404 {object} response.Response{error=string}
// @Failure 500 {object} response.Response{error=string}
// @Router /api/v1/media-mentions/{id} [put]
func (h *MediaMentionHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid media mention ID")
	}

	existing, err := h.service.GetByID(uint(id))
	if err != nil {
		return response.NotFound(c, "Media mention not found")
	}

	var req mediamention.MediaMention
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	if req.Title != "" {
		if len(req.Title) < 2 {
			return response.BadRequest(c, "Title must be at least 2 characters")
		}
		existing.Title = req.Title
	}

	bodyMap := make(map[string]interface{})
	if err := c.BodyParser(&bodyMap); err == nil {
		if _, ok := bodyMap["link"]; ok {
			if req.Link != "" {
				if err := validation.ValidateURL(req.Link); err != nil {
					return response.BadRequest(c, "Invalid link URL: "+err.Error())
				}
			}
			existing.Link = req.Link
		}
		if _, ok := bodyMap["displayOrder"]; ok {
			existing.DisplayOrder = req.DisplayOrder
		}
		_, reportIDTouched := bodyMap["reportId"]
		_, reportLinkTextTouched := bodyMap["reportLinkText"]
		if reportIDTouched {
			existing.ReportID = req.ReportID
		}
		if reportLinkTextTouched {
			existing.ReportLinkText = req.ReportLinkText
		}
		if reportIDTouched || reportLinkTextTouched {
			if err := h.validateReportReference(existing.ReportID, existing.ReportLinkText); err != nil {
				return response.BadRequest(c, err.Error())
			}
		}
	}

	if err := h.service.Update(uint(id), existing); err != nil {
		return response.InternalError(c, "Failed to update media mention")
	}

	return response.Success(c, existing)
}

// Delete godoc
// @Summary Delete media mention
// @Description Delete a media mention by ID. Requires admin role.
// @Tags MediaMentions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Media Mention ID"
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 400 {object} response.Response{error=string}
// @Failure 401 {object} response.Response{error=string}
// @Failure 403 {object} response.Response{error=string}
// @Failure 404 {object} response.Response{error=string}
// @Failure 500 {object} response.Response{error=string}
// @Router /api/v1/media-mentions/{id} [delete]
func (h *MediaMentionHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid media mention ID")
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Media mention not found")
		}
		return response.InternalError(c, "Failed to delete media mention")
	}

	return response.Success(c, fiber.Map{
		"message": "Media mention deleted successfully",
	})
}

// UploadImage godoc
// @Summary Upload media mention logo image
// @Description Upload or replace the logo image for a media mention. Requires admin or editor role.
// @Tags MediaMentions
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Media Mention ID"
// @Param image formData file true "Image file (max 10MB, allowed types: JPEG, PNG, WebP, GIF)"
// @Success 200 {object} response.Response{data=mediamention.MediaMention}
// @Failure 400 {object} response.Response{error=string}
// @Failure 401 {object} response.Response{error=string}
// @Failure 403 {object} response.Response{error=string}
// @Failure 404 {object} response.Response{error=string}
// @Failure 500 {object} response.Response{error=string}
// @Router /api/v1/media-mentions/{id}/image [post]
func (h *MediaMentionHandler) UploadImage(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid media mention ID")
	}

	file, err := c.FormFile("image")
	if err != nil {
		return response.BadRequest(c, "No image file provided")
	}

	if err := validation.ValidateImageFile(file); err != nil {
		return response.BadRequest(c, err.Error())
	}

	updated, err := h.service.UploadImage(uint(id), file)
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Media mention not found")
		}
		return response.InternalError(c, "Failed to upload image: "+err.Error())
	}

	return response.Success(c, updated)
}

// DeleteImage godoc
// @Summary Delete media mention logo image
// @Description Delete the logo image of a media mention. Requires admin or editor role.
// @Tags MediaMentions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Media Mention ID"
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 400 {object} response.Response{error=string}
// @Failure 401 {object} response.Response{error=string}
// @Failure 403 {object} response.Response{error=string}
// @Failure 404 {object} response.Response{error=string}
// @Failure 500 {object} response.Response{error=string}
// @Router /api/v1/media-mentions/{id}/image [delete]
func (h *MediaMentionHandler) DeleteImage(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid media mention ID")
	}

	if err := h.service.DeleteImage(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Media mention not found")
		}
		if err.Error() == "media mention has no image to delete" {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "Failed to delete image: "+err.Error())
	}

	return response.Success(c, fiber.Map{
		"message": "Image deleted successfully",
	})
}
