package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/mediamention"
	"github.com/healthcare-market-research/backend/internal/domain/report"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type mockMediaMentionService struct {
	getAllFunc  func(page, limit int, search string) ([]mediamention.MediaMention, int64, error)
	getByIDFunc func(id uint) (*mediamention.MediaMention, error)
	createFunc  func(mention *mediamention.MediaMention) error
	updateFunc  func(id uint, mention *mediamention.MediaMention) error
	deleteFunc  func(id uint) error
}

func (m *mockMediaMentionService) GetAll(page, limit int, search string) ([]mediamention.MediaMention, int64, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(page, limit, search)
	}
	return []mediamention.MediaMention{}, 0, nil
}
func (m *mockMediaMentionService) GetByID(id uint) (*mediamention.MediaMention, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &mediamention.MediaMention{ID: id, Title: "Test"}, nil
}
func (m *mockMediaMentionService) Create(mention *mediamention.MediaMention) error {
	if m.createFunc != nil {
		return m.createFunc(mention)
	}
	return nil
}
func (m *mockMediaMentionService) Update(id uint, mention *mediamention.MediaMention) error {
	if m.updateFunc != nil {
		return m.updateFunc(id, mention)
	}
	return nil
}
func (m *mockMediaMentionService) Delete(id uint) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}
func (m *mockMediaMentionService) UploadImage(id uint, file *multipart.FileHeader) (*mediamention.MediaMention, error) {
	return nil, nil
}
func (m *mockMediaMentionService) DeleteImage(id uint) error { return nil }

type mockReportLookup struct {
	getByIDFunc func(id uint) (*report.Report, error)
}

func (m *mockReportLookup) GetByID(id uint) (*report.Report, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &report.Report{ID: id, Title: "VFX Market Report", Slug: "vfx-market-report", Status: "published"}, nil
}

func TestMediaMentionHandler_Create_WithPublishedReport_Succeeds(t *testing.T) {
	mockService := &mockMediaMentionService{}
	mockReports := &mockReportLookup{}
	h := NewMediaMentionHandler(mockService, mockReports)

	app := fiber.New()
	app.Post("/media-mentions", h.Create)

	body, _ := json.Marshal(map[string]interface{}{
		"title":          "Forbes",
		"reportId":       1,
		"reportLinkText": "Global VFX Market",
	})
	req := httptest.NewRequest("POST", "/media-mentions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestMediaMentionHandler_Create_WithUnpublishedReport_Rejected(t *testing.T) {
	mockService := &mockMediaMentionService{}
	mockReports := &mockReportLookup{
		getByIDFunc: func(id uint) (*report.Report, error) {
			return &report.Report{ID: id, Status: "draft"}, nil
		},
	}
	h := NewMediaMentionHandler(mockService, mockReports)

	app := fiber.New()
	app.Post("/media-mentions", h.Create)

	body, _ := json.Marshal(map[string]interface{}{
		"title":          "Forbes",
		"reportId":       1,
		"reportLinkText": "Global VFX Market",
	})
	req := httptest.NewRequest("POST", "/media-mentions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}

	var result response.Response
	bodyBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &result)
	if result.Success {
		t.Error("expected success=false")
	}
}

func TestMediaMentionHandler_Create_ReportIDWithoutLinkText_Rejected(t *testing.T) {
	h := NewMediaMentionHandler(&mockMediaMentionService{}, &mockReportLookup{})

	app := fiber.New()
	app.Post("/media-mentions", h.Create)

	body, _ := json.Marshal(map[string]interface{}{
		"title":    "Forbes",
		"reportId": 1,
	})
	req := httptest.NewRequest("POST", "/media-mentions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestMediaMentionHandler_Create_NoReportFields_Succeeds(t *testing.T) {
	h := NewMediaMentionHandler(&mockMediaMentionService{}, &mockReportLookup{})

	app := fiber.New()
	app.Post("/media-mentions", h.Create)

	body, _ := json.Marshal(map[string]interface{}{"title": "Forbes"})
	req := httptest.NewRequest("POST", "/media-mentions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestMediaMentionHandler_Update_UnlinksReportWhenExplicitlyNulled(t *testing.T) {
	existing := &mediamention.MediaMention{
		ID: 1, Title: "Forbes",
		ReportID:       func() *uint { id := uint(1); return &id }(),
		ReportLinkText: "Old Caption",
	}
	var savedMention *mediamention.MediaMention
	mockService := &mockMediaMentionService{
		getByIDFunc: func(id uint) (*mediamention.MediaMention, error) { return existing, nil },
		updateFunc: func(id uint, mention *mediamention.MediaMention) error {
			savedMention = mention
			return nil
		},
	}
	h := NewMediaMentionHandler(mockService, &mockReportLookup{})

	app := fiber.New()
	app.Put("/media-mentions/:id", h.Update)

	body, _ := json.Marshal(map[string]interface{}{
		"reportId":       nil,
		"reportLinkText": "",
	})
	req := httptest.NewRequest("PUT", "/media-mentions/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if savedMention == nil {
		t.Fatal("expected Update to be called")
	}
	if savedMention.ReportID != nil {
		t.Error("expected ReportID to be cleared")
	}
}

var _ = time.Now
var _ = errors.New
