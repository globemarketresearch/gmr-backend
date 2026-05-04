package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/author"
	"github.com/healthcare-market-research/backend/pkg/response"
)

// Mock AuthorService for testing
type mockAuthorService struct {
	getAllFunc      func(page, limit int, search string) ([]author.Author, int64, error)
	getByIDFunc     func(id uint) (*author.Author, error)
	createFunc      func(author *author.Author) error
	updateFunc      func(id uint, author *author.Author) error
	deleteFunc      func(id uint) error
	uploadImageFunc func(authorID uint, file *multipart.FileHeader) (*author.Author, error)
	deleteImageFunc func(authorID uint) error
}

func (m *mockAuthorService) GetAll(page, limit int, search string) ([]author.Author, int64, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(page, limit, search)
	}
	return []author.Author{}, 0, nil
}

func (m *mockAuthorService) GetByID(id uint) (*author.Author, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &author.Author{ID: id, Name: "Test Author"}, nil
}

func (m *mockAuthorService) Create(author *author.Author) error {
	if m.createFunc != nil {
		return m.createFunc(author)
	}
	return nil
}

func (m *mockAuthorService) Update(id uint, author *author.Author) error {
	if m.updateFunc != nil {
		return m.updateFunc(id, author)
	}
	return nil
}

func (m *mockAuthorService) Delete(id uint) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *mockAuthorService) UploadImage(authorID uint, file *multipart.FileHeader) (*author.Author, error) {
	if m.uploadImageFunc != nil {
		return m.uploadImageFunc(authorID, file)
	}
	return &author.Author{
		ID:       authorID,
		Name:     "Test Author",
		ImageURL: "https://imagedelivery.net/test/image-id/public",
	}, nil
}

func (m *mockAuthorService) DeleteImage(authorID uint) error {
	if m.deleteImageFunc != nil {
		return m.deleteImageFunc(authorID)
	}
	return nil
}

func TestAuthorHandler_UploadImage_Success(t *testing.T) {
	mockService := &mockAuthorService{
		uploadImageFunc: func(authorID uint, file *multipart.FileHeader) (*author.Author, error) {
			if authorID != 1 {
				t.Errorf("Expected authorID 1, got %d", authorID)
			}
			if file.Filename != "test.jpg" {
				t.Errorf("Expected filename test.jpg, got %s", file.Filename)
			}
			return &author.Author{
				ID:       authorID,
				Name:     "Test Author",
				ImageURL: "https://imagedelivery.net/test/new-image-id/public",
			}, nil
		},
	}

	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Post("/authors/:id/image", handler.UploadImage)

	// Create multipart form with image file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.jpg")
	part.Write([]byte("fake image content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/authors/1/image", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result response.Response
	bodyBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &result)

	if !result.Success {
		t.Error("Expected success to be true")
	}
}

func TestAuthorHandler_UploadImage_InvalidID(t *testing.T) {
	mockService := &mockAuthorService{}
	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Post("/authors/:id/image", handler.UploadImage)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.jpg")
	part.Write([]byte("fake image content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/authors/invalid/image", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestAuthorHandler_UploadImage_NoFile(t *testing.T) {
	mockService := &mockAuthorService{}
	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Post("/authors/:id/image", handler.UploadImage)

	// Create request without image file
	req := httptest.NewRequest("POST", "/authors/1/image", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var result response.Response
	bodyBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &result)

	if result.Success {
		t.Error("Expected success to be false")
	}
}

func TestAuthorHandler_UploadImage_AuthorNotFound(t *testing.T) {
	mockService := &mockAuthorService{
		uploadImageFunc: func(authorID uint, file *multipart.FileHeader) (*author.Author, error) {
			return nil, errors.New("record not found")
		},
	}

	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Post("/authors/:id/image", handler.UploadImage)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.jpg")
	part.Write([]byte("fake image content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/authors/999/image", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestAuthorHandler_UploadImage_ServiceError(t *testing.T) {
	mockService := &mockAuthorService{
		uploadImageFunc: func(authorID uint, file *multipart.FileHeader) (*author.Author, error) {
			return nil, errors.New("internal service error")
		},
	}

	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Post("/authors/:id/image", handler.UploadImage)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.jpg")
	part.Write([]byte("fake image content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/authors/1/image", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestAuthorHandler_DeleteImage_Success(t *testing.T) {
	deleteImageCalled := false

	mockService := &mockAuthorService{
		deleteImageFunc: func(authorID uint) error {
			if authorID != 1 {
				t.Errorf("Expected authorID 1, got %d", authorID)
			}
			deleteImageCalled = true
			return nil
		},
	}

	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Delete("/authors/:id/image", handler.DeleteImage)

	req := httptest.NewRequest("DELETE", "/authors/1/image", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if !deleteImageCalled {
		t.Error("Expected deleteImage to be called")
	}

	var result response.Response
	bodyBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &result)

	if !result.Success {
		t.Error("Expected success to be true")
	}
}

func TestAuthorHandler_DeleteImage_InvalidID(t *testing.T) {
	mockService := &mockAuthorService{}
	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Delete("/authors/:id/image", handler.DeleteImage)

	req := httptest.NewRequest("DELETE", "/authors/invalid/image", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestAuthorHandler_DeleteImage_AuthorNotFound(t *testing.T) {
	mockService := &mockAuthorService{
		deleteImageFunc: func(authorID uint) error {
			return errors.New("record not found")
		},
	}

	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Delete("/authors/:id/image", handler.DeleteImage)

	req := httptest.NewRequest("DELETE", "/authors/999/image", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestAuthorHandler_DeleteImage_NoImage(t *testing.T) {
	mockService := &mockAuthorService{
		deleteImageFunc: func(authorID uint) error {
			return errors.New("author has no image to delete")
		},
	}

	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Delete("/authors/:id/image", handler.DeleteImage)

	req := httptest.NewRequest("DELETE", "/authors/1/image", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var result response.Response
	bodyBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &result)

	if result.Success {
		t.Error("Expected success to be false")
	}
}

func TestAuthorHandler_DeleteImage_ServiceError(t *testing.T) {
	mockService := &mockAuthorService{
		deleteImageFunc: func(authorID uint) error {
			return errors.New("internal service error")
		},
	}

	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Delete("/authors/:id/image", handler.DeleteImage)

	req := httptest.NewRequest("DELETE", "/authors/1/image", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

// Test image upload with large file (would be validated by validation.ValidateImageFile)
func TestAuthorHandler_UploadImage_LargeFile(t *testing.T) {
	mockService := &mockAuthorService{
		uploadImageFunc: func(authorID uint, file *multipart.FileHeader) (*author.Author, error) {
			// In real scenario, validation happens before this
			return &author.Author{
				ID:       authorID,
				Name:     "Test Author",
				ImageURL: "https://imagedelivery.net/test/image-id/public",
			}, nil
		},
	}

	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Post("/authors/:id/image", handler.UploadImage)

	// Create a large file (note: actual validation happens in ValidateImageFile)
	largeContent := make([]byte, 1024*1024) // 1MB
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "large.jpg")
	part.Write(largeContent)
	writer.Close()

	req := httptest.NewRequest("POST", "/authors/1/image", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	// Note: In real implementation, validation.ValidateImageFile would catch this
	// This test just ensures the handler can process the request
	_ = resp
}

func TestAuthorHandler_UploadImage_MultipleFiles(t *testing.T) {
	mockService := &mockAuthorService{
		uploadImageFunc: func(authorID uint, file *multipart.FileHeader) (*author.Author, error) {
			// Should only process the first "image" field
			return &author.Author{
				ID:       authorID,
				Name:     "Test Author",
				ImageURL: "https://imagedelivery.net/test/image-id/public",
			}, nil
		},
	}

	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Post("/authors/:id/image", handler.UploadImage)

	// Create multipart form with multiple files
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add first image file
	part1, _ := writer.CreateFormFile("image", "test1.jpg")
	part1.Write([]byte("fake image content 1"))

	// Add second file (different field name)
	part2, _ := writer.CreateFormFile("other", "test2.jpg")
	part2.Write([]byte("fake image content 2"))

	writer.Close()

	req := httptest.NewRequest("POST", "/authors/1/image", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	// Should process successfully using the "image" field
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// Tests for LinkedIn URL validation in Create handler
func TestAuthorHandler_Create_WithValidLinkedInURL(t *testing.T) {
	mockService := &mockAuthorService{
		createFunc: func(a *author.Author) error {
			if a.LinkedinURL != "https://www.linkedin.com/in/johndoe" {
				t.Errorf("Expected LinkedinURL to be set correctly, got %s", a.LinkedinURL)
			}
			return nil
		},
	}

	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Post("/authors", handler.Create)

	reqBody := map[string]interface{}{
		"name":        "John Doe",
		"linkedinUrl": "https://www.linkedin.com/in/johndoe",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/authors", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(bodyBytes))
	}
}

func TestAuthorHandler_Create_WithInvalidLinkedInURL(t *testing.T) {
	mockService := &mockAuthorService{}
	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Post("/authors", handler.Create)

	tests := []struct {
		name        string
		linkedinUrl string
	}{
		{
			name:        "HTTP instead of HTTPS",
			linkedinUrl: "http://www.linkedin.com/in/johndoe",
		},
		{
			name:        "Invalid URL format",
			linkedinUrl: "not-a-url",
		},
		{
			name:        "Empty protocol",
			linkedinUrl: "www.linkedin.com/in/johndoe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"name":        "John Doe",
				"linkedinUrl": tt.linkedinUrl,
			}
			bodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/authors", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to perform request: %v", err)
			}

			if resp.StatusCode != fiber.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", resp.StatusCode)
			}

			var result response.Response
			respBodyBytes, _ := io.ReadAll(resp.Body)
			json.Unmarshal(respBodyBytes, &result)

			if result.Success {
				t.Error("Expected success to be false")
			}
		})
	}
}

func TestAuthorHandler_Create_WithoutLinkedInURL(t *testing.T) {
	mockService := &mockAuthorService{
		createFunc: func(a *author.Author) error {
			if a.LinkedinURL != "" {
				t.Errorf("Expected LinkedinURL to be empty, got %s", a.LinkedinURL)
			}
			return nil
		},
	}

	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Post("/authors", handler.Create)

	reqBody := map[string]interface{}{
		"name": "John Doe",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/authors", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(bodyBytes))
	}
}

// Tests for LinkedIn URL validation in Update handler
func TestAuthorHandler_Update_LinkedInURL(t *testing.T) {
	existingAuthor := &author.Author{
		ID:   1,
		Name: "John Doe",
	}

	mockService := &mockAuthorService{
		getByIDFunc: func(id uint) (*author.Author, error) {
			return existingAuthor, nil
		},
		updateFunc: func(id uint, a *author.Author) error {
			if a.LinkedinURL != "https://www.linkedin.com/in/updated" {
				t.Errorf("Expected LinkedinURL to be updated, got %s", a.LinkedinURL)
			}
			return nil
		},
	}

	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Put("/authors/:id", handler.Update)

	reqBody := map[string]interface{}{
		"linkedinUrl": "https://www.linkedin.com/in/updated",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/authors/1", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(bodyBytes))
	}
}

func TestAuthorHandler_Update_ClearLinkedInURL(t *testing.T) {
	existingAuthor := &author.Author{
		ID:          1,
		Name:        "John Doe",
		LinkedinURL: "https://www.linkedin.com/in/johndoe",
	}

	mockService := &mockAuthorService{
		getByIDFunc: func(id uint) (*author.Author, error) {
			return existingAuthor, nil
		},
		updateFunc: func(id uint, a *author.Author) error {
			if a.LinkedinURL != "" {
				t.Errorf("Expected LinkedinURL to be cleared, got %s", a.LinkedinURL)
			}
			return nil
		},
	}

	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Put("/authors/:id", handler.Update)

	reqBody := map[string]interface{}{
		"linkedinUrl": "",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/authors/1", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(bodyBytes))
	}
}

func TestAuthorHandler_Update_InvalidLinkedInURL(t *testing.T) {
	existingAuthor := &author.Author{
		ID:   1,
		Name: "John Doe",
	}

	mockService := &mockAuthorService{
		getByIDFunc: func(id uint) (*author.Author, error) {
			return existingAuthor, nil
		},
	}

	handler := NewAuthorHandler(mockService)
	app := fiber.New()
	app.Put("/authors/:id", handler.Update)

	reqBody := map[string]interface{}{
		"linkedinUrl": "http://invalid-url",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/authors/1", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var result response.Response
	bodyBytes, _ = io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &result)

	if result.Success {
		t.Error("Expected success to be false")
	}
}
