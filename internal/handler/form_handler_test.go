package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/form"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockFormService is a mock implementation of FormService
type MockFormService struct {
	mock.Mock
}

func (m *MockFormService) Create(req *form.CreateSubmissionRequest) (*form.SubmissionResponse, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*form.SubmissionResponse), args.Error(1)
}

func (m *MockFormService) GetAll(query form.GetSubmissionsQuery) ([]form.FormSubmission, int64, error) {
	args := m.Called(query)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]form.FormSubmission), int64(args.Int(1)), args.Error(2)
}

func (m *MockFormService) GetByID(id uint) (*form.FormSubmission, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*form.FormSubmission), args.Error(1)
}

func (m *MockFormService) GetByCategory(category string, page, limit int) ([]form.FormSubmission, int64, error) {
	args := m.Called(category, page, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]form.FormSubmission), int64(args.Int(1)), args.Error(2)
}

func (m *MockFormService) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockFormService) BulkDelete(ids []uint) (int64, error) {
	args := m.Called(ids)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockFormService) GetStats() (*form.SubmissionStats, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*form.SubmissionStats), args.Error(1)
}

func (m *MockFormService) UpdateStatus(id uint, status form.FormStatus, processedBy *uint) error {
	args := m.Called(id, status, processedBy)
	return args.Error(0)
}

func setupTestApp(handler *FormHandler) *fiber.App {
	app := fiber.New()
	app.Get("/api/v1/forms/submissions", handler.GetAll)
	app.Get("/api/v1/forms/submissions/:id", handler.GetByID)
	app.Post("/api/v1/forms/submissions", handler.Create)
	return app
}

func TestFormHandler_GetByID(t *testing.T) {
	mockService := new(MockFormService)
	handler := NewFormHandler(mockService)
	app := setupTestApp(handler)

	t.Run("Successfully retrieve submission by ID", func(t *testing.T) {
		expectedSubmission := &form.FormSubmission{
			ID:       1,
			Category: form.CategoryContact,
			Status:   form.StatusPending,
			Data: form.FormData{
				"fullName": "John Doe",
				"email":    "john@example.com",
				"company":  "Test Corp",
				"subject":  "Test",
				"message":  "Message",
			},
		}

		mockService.On("GetByID", uint(1)).Return(expectedSubmission, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/forms/submissions/1", nil)
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

	t.Run("Return 404 for non-existent ID", func(t *testing.T) {
		mockService.On("GetByID", uint(999)).Return(nil, gorm.ErrRecordNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/forms/submissions/999", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Return 400 for invalid ID format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/forms/submissions/invalid", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var response map[string]interface{}
		json.Unmarshal(body, &response)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(string), "Invalid submission ID")
	})
}

func TestFormHandler_Create(t *testing.T) {
	mockService := new(MockFormService)
	handler := NewFormHandler(mockService)
	app := setupTestApp(handler)

	t.Run("Create submission successfully", func(t *testing.T) {
		requestBody := form.CreateSubmissionRequest{
			Category: form.CategoryContact,
			Data: form.FormData{
				"fullName": "John Doe",
				"email":    "john@example.com",
				"company":  "Test Corp",
				"subject":  "Test Subject",
				"message":  "Test Message",
			},
		}

		expectedResponse := &form.SubmissionResponse{
			Success:      true,
			SubmissionID: uint(1),
			Category:     form.CategoryContact,
			Message:      "Form submitted successfully",
		}

		mockService.On("Create", mock.AnythingOfType("*form.CreateSubmissionRequest")).Return(expectedResponse, nil).Once()

		bodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/forms/submissions", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var response form.SubmissionResponse
		json.Unmarshal(body, &response)

		assert.True(t, response.Success)
		assert.Equal(t, uint(1), response.SubmissionID)

		mockService.AssertExpectations(t)
	})
}
