package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockReportImageService is a mock implementation of ReportImageService
type MockReportImageService struct {
	mock.Mock
}

func (m *MockReportImageService) UploadImage(reportID uint, file *multipart.FileHeader, title string, uploadedBy uint) (*report.ReportImage, error) {
	args := m.Called(reportID, file, title, uploadedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*report.ReportImage), args.Error(1)
}

func (m *MockReportImageService) UpdateImageMetadata(imageID uint, title *string, isActive *bool) (*report.ReportImage, error) {
	args := m.Called(imageID, title, isActive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*report.ReportImage), args.Error(1)
}

func (m *MockReportImageService) DeleteImage(imageID uint) error {
	args := m.Called(imageID)
	return args.Error(0)
}

func (m *MockReportImageService) GetImagesByReport(reportID uint, activeOnly bool) ([]report.ReportImage, error) {
	args := m.Called(reportID, activeOnly)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]report.ReportImage), args.Error(1)
}

func (m *MockReportImageService) GetImageByID(imageID uint) (*report.ReportImage, error) {
	args := m.Called(imageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*report.ReportImage), args.Error(1)
}

func setupReportImageTestApp(handler *ReportImageHandler) *fiber.App {
	app := fiber.New()

	// Setup middleware to set user_id
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(5))
		return c.Next()
	})

	app.Post("/api/v1/reports/:reportId/images", handler.UploadImage)
	app.Get("/api/v1/reports/:reportId/images", handler.ListImages)
	app.Get("/api/v1/reports/images/:imageId", handler.GetByID)
	app.Patch("/api/v1/reports/images/:imageId", handler.UpdateMetadata)
	app.Delete("/api/v1/reports/images/:imageId", handler.DeleteImage)

	return app
}

// Helper to create multipart form with file
func createMultipartRequest(filename string, content string, extraFields map[string]string) (*http.Request, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file with valid PNG header to pass validation
	part, _ := writer.CreateFormFile("image", filename)
	// Write a minimal valid PNG header
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	part.Write(pngHeader)
	part.Write([]byte(content))

	// Add extra fields
	for key, value := range extraFields {
		writer.WriteField(key, value)
	}

	writer.Close()
	return httptest.NewRequest(http.MethodPost, "/api/v1/reports/1/images", body), writer.FormDataContentType()
}

func TestReportImageHandler_UploadImage(t *testing.T) {
	mockService := new(MockReportImageService)
	handler := NewReportImageHandler(mockService)
	app := setupReportImageTestApp(handler)

	t.Run("Successfully upload image with title", func(t *testing.T) {
		userID := uint(5)
		expectedImage := &report.ReportImage{
			ID:         1,
			ReportID:   1,
			ImageURL:   "https://imagedelivery.net/test/image-id/public",
			Title:      "Test Chart",
			IsActive:   true,
			UploadedBy: &userID,
		}

		mockService.On("UploadImage", uint(1), mock.Anything, "Test Chart", uint(5)).Return(expectedImage, nil).Once()

		req, contentType := createMultipartRequest("chart.png", "fake image content", map[string]string{"title": "Test Chart"})
		req.Header.Set("Content-Type", contentType)

		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var response map[string]interface{}
		json.Unmarshal(body, &response)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["id"].(float64))
		assert.Equal(t, "Test Chart", data["title"].(string))

		mockService.AssertExpectations(t)
	})

	t.Run("Successfully upload image without title", func(t *testing.T) {
		userID := uint(5)
		expectedImage := &report.ReportImage{
			ID:         2,
			ReportID:   1,
			ImageURL:   "https://imagedelivery.net/test/image-id-2/public",
			Title:      "",
			IsActive:   true,
			UploadedBy: &userID,
		}

		mockService.On("UploadImage", uint(1), mock.Anything, "", uint(5)).Return(expectedImage, nil).Once()

		req, contentType := createMultipartRequest("chart.png", "fake image content", map[string]string{})
		req.Header.Set("Content-Type", contentType)

		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Return 400 for invalid report ID", func(t *testing.T) {
		req, contentType := createMultipartRequest("chart.png", "fake image content", map[string]string{})
		req.URL.Path = "/api/v1/reports/invalid/images"
		req.Header.Set("Content-Type", contentType)

		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Return 404 when report not found", func(t *testing.T) {
		mockService.On("UploadImage", uint(999), mock.Anything, "", uint(5)).Return(nil, errors.New("report not found")).Once()

		req, contentType := createMultipartRequest("chart.png", "fake image content", map[string]string{})
		req.URL.Path = "/api/v1/reports/999/images"
		req.Header.Set("Content-Type", contentType)

		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Return 400 for title too short", func(t *testing.T) {
		req, contentType := createMultipartRequest("chart.png", "fake image content", map[string]string{"title": "A"})
		req.Header.Set("Content-Type", contentType)

		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var response map[string]interface{}
		json.Unmarshal(body, &response)

		assert.Contains(t, response["error"].(string), "between 2 and 255")
	})
}

func TestReportImageHandler_ListImages(t *testing.T) {
	mockService := new(MockReportImageService)
	handler := NewReportImageHandler(mockService)
	app := setupReportImageTestApp(handler)

	t.Run("Successfully list all images", func(t *testing.T) {
		userID := uint(5)
		expectedImages := []report.ReportImage{
			{ID: 1, ReportID: 1, Title: "Image 1", IsActive: true, UploadedBy: &userID},
			{ID: 2, ReportID: 1, Title: "Image 2", IsActive: false, UploadedBy: &userID},
		}

		mockService.On("GetImagesByReport", uint(1), false).Return(expectedImages, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/1/images", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var response map[string]interface{}
		json.Unmarshal(body, &response)

		assert.True(t, response["success"].(bool))
		data := response["data"].([]interface{})
		assert.Len(t, data, 2)

		mockService.AssertExpectations(t)
	})

	t.Run("Successfully list active images only", func(t *testing.T) {
		userID := uint(5)
		expectedImages := []report.ReportImage{
			{ID: 1, ReportID: 1, Title: "Image 1", IsActive: true, UploadedBy: &userID},
		}

		mockService.On("GetImagesByReport", uint(1), true).Return(expectedImages, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/1/images?active=true", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var response map[string]interface{}
		json.Unmarshal(body, &response)

		assert.True(t, response["success"].(bool))
		data := response["data"].([]interface{})
		assert.Len(t, data, 1)

		mockService.AssertExpectations(t)
	})

	t.Run("Return 400 for invalid report ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/invalid/images", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestReportImageHandler_GetByID(t *testing.T) {
	mockService := new(MockReportImageService)
	handler := NewReportImageHandler(mockService)
	app := setupReportImageTestApp(handler)

	t.Run("Successfully retrieve image by ID", func(t *testing.T) {
		userID := uint(5)
		expectedImage := &report.ReportImage{
			ID:         1,
			ReportID:   1,
			ImageURL:   "https://imagedelivery.net/test/image-id/public",
			Title:      "Test Image",
			IsActive:   true,
			UploadedBy: &userID,
		}

		mockService.On("GetImageByID", uint(1)).Return(expectedImage, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/images/1", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var response map[string]interface{}
		json.Unmarshal(body, &response)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["id"].(float64))

		mockService.AssertExpectations(t)
	})

	t.Run("Return 404 for non-existent image", func(t *testing.T) {
		mockService.On("GetImageByID", uint(999)).Return(nil, errors.New("image not found")).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/images/999", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Return 400 for invalid image ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/images/invalid", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestReportImageHandler_UpdateMetadata(t *testing.T) {
	mockService := new(MockReportImageService)
	handler := NewReportImageHandler(mockService)
	app := setupReportImageTestApp(handler)

	t.Run("Successfully update title", func(t *testing.T) {
		newTitle := "Updated Title"
		userID := uint(5)
		expectedImage := &report.ReportImage{
			ID:         1,
			ReportID:   1,
			ImageURL:   "https://imagedelivery.net/test/image-id/public",
			Title:      newTitle,
			IsActive:   true,
			UploadedBy: &userID,
		}

		mockService.On("UpdateImageMetadata", uint(1), &newTitle, (*bool)(nil)).Return(expectedImage, nil).Once()

		reqBody := UpdateImageMetadataRequest{
			Title: &newTitle,
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/reports/images/1", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var response map[string]interface{}
		json.Unmarshal(body, &response)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, newTitle, data["title"].(string))

		mockService.AssertExpectations(t)
	})

	t.Run("Successfully update is_active", func(t *testing.T) {
		isActive := false
		userID := uint(5)
		expectedImage := &report.ReportImage{
			ID:         1,
			ReportID:   1,
			ImageURL:   "https://imagedelivery.net/test/image-id/public",
			Title:      "Original Title",
			IsActive:   false,
			UploadedBy: &userID,
		}

		mockService.On("UpdateImageMetadata", uint(1), (*string)(nil), &isActive).Return(expectedImage, nil).Once()

		reqBody := UpdateImageMetadataRequest{
			IsActive: &isActive,
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/reports/images/1", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Return 404 for non-existent image", func(t *testing.T) {
		newTitle := "Updated Title"
		mockService.On("UpdateImageMetadata", uint(999), &newTitle, (*bool)(nil)).Return(nil, errors.New("image not found")).Once()

		reqBody := UpdateImageMetadataRequest{
			Title: &newTitle,
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/reports/images/999", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Return 400 for title too short", func(t *testing.T) {
		shortTitle := "A"
		reqBody := UpdateImageMetadataRequest{
			Title: &shortTitle,
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/reports/images/1", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var response map[string]interface{}
		json.Unmarshal(body, &response)

		assert.Contains(t, response["error"].(string), "between 2 and 255")
	})
}

func TestReportImageHandler_DeleteImage(t *testing.T) {
	mockService := new(MockReportImageService)
	handler := NewReportImageHandler(mockService)
	app := setupReportImageTestApp(handler)

	t.Run("Successfully delete image", func(t *testing.T) {
		mockService.On("DeleteImage", uint(1)).Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/reports/images/1", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var response map[string]interface{}
		json.Unmarshal(body, &response)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, "Image deleted successfully", data["message"].(string))

		mockService.AssertExpectations(t)
	})

	t.Run("Return 404 for non-existent image", func(t *testing.T) {
		mockService.On("DeleteImage", uint(999)).Return(errors.New("image not found")).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/reports/images/999", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Return 400 for invalid image ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/reports/images/invalid", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
