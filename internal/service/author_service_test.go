package service

import (
	"bytes"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/healthcare-market-research/backend/internal/domain/author"
)

// Mock CloudflareImagesService for testing
type mockCloudflareService struct {
	uploadFunc      func(file *multipart.FileHeader, metadata map[string]string) (string, error)
	deleteFunc      func(imageURL string) error
	extractIDFunc   func(imageURL string) (string, error)
}

func (m *mockCloudflareService) Upload(file *multipart.FileHeader, metadata map[string]string) (string, error) {
	if m.uploadFunc != nil {
		return m.uploadFunc(file, metadata)
	}
	return "https://imagedelivery.net/test/image-id-123/public", nil
}

func (m *mockCloudflareService) Delete(imageURL string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(imageURL)
	}
	return nil
}

func (m *mockCloudflareService) ExtractImageID(imageURL string) (string, error) {
	if m.extractIDFunc != nil {
		return m.extractIDFunc(imageURL)
	}
	return "image-id-123", nil
}

// Mock AuthorRepository for testing
type mockAuthorRepository struct {
	getByIDFunc             func(id uint) (*author.Author, error)
	updateFunc              func(author *author.Author) error
	deleteFunc              func(id uint) error
	isReferencedInReportsFunc func(id uint) (bool, error)
}

func (m *mockAuthorRepository) GetByID(id uint) (*author.Author, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &author.Author{ID: id, Name: "Test Author"}, nil
}

func (m *mockAuthorRepository) Update(author *author.Author) error {
	if m.updateFunc != nil {
		return m.updateFunc(author)
	}
	return nil
}

func (m *mockAuthorRepository) Delete(id uint) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *mockAuthorRepository) IsReferencedInReports(id uint) (bool, error) {
	if m.isReferencedInReportsFunc != nil {
		return m.isReferencedInReportsFunc(id)
	}
	return false, nil
}

func (m *mockAuthorRepository) GetAll(page, limit int, search string) ([]author.Author, int64, error) {
	return nil, 0, nil
}

func (m *mockAuthorRepository) Create(author *author.Author) error {
	return nil
}

// Helper to create a test file header
func createTestFileHeader(filename string, content string) *multipart.FileHeader {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filename)
	part.Write([]byte(content))
	writer.Close()

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.ParseMultipartForm(10 << 20)
	_, fileHeader, _ := req.FormFile("file")
	return fileHeader
}

func TestAuthorService_UploadImage_Success(t *testing.T) {
	mockRepo := &mockAuthorRepository{
		getByIDFunc: func(id uint) (*author.Author, error) {
			return &author.Author{
				ID:       id,
				Name:     "Test Author",
				ImageURL: "",
			}, nil
		},
		updateFunc: func(auth *author.Author) error {
			if auth.ImageURL != "https://imagedelivery.net/test/new-image-id/public" {
				t.Errorf("Expected imageURL to be updated")
			}
			return nil
		},
	}

	mockCloudflare := &mockCloudflareService{
		uploadFunc: func(file *multipart.FileHeader, metadata map[string]string) (string, error) {
			if metadata["author_id"] != "1" {
				t.Errorf("Expected author_id metadata to be '1', got '%s'", metadata["author_id"])
			}
			return "https://imagedelivery.net/test/new-image-id/public", nil
		},
	}

	service := NewAuthorService(mockRepo, mockCloudflare)
	fileHeader := createTestFileHeader("test.jpg", "fake image content")

	updatedAuthor, err := service.UploadImage(1, fileHeader)
	if err != nil {
		t.Errorf("UploadImage() error = %v, want nil", err)
	}

	if updatedAuthor.ImageURL != "https://imagedelivery.net/test/new-image-id/public" {
		t.Errorf("Expected imageURL to be set, got %s", updatedAuthor.ImageURL)
	}
}

func TestAuthorService_UploadImage_ReplaceExisting(t *testing.T) {
	oldImageURL := "https://imagedelivery.net/test/old-image-id/public"
	deleteCalled := false

	mockRepo := &mockAuthorRepository{
		getByIDFunc: func(id uint) (*author.Author, error) {
			return &author.Author{
				ID:       id,
				Name:     "Test Author",
				ImageURL: oldImageURL,
			}, nil
		},
		updateFunc: func(auth *author.Author) error {
			return nil
		},
	}

	mockCloudflare := &mockCloudflareService{
		deleteFunc: func(imageURL string) error {
			if imageURL != oldImageURL {
				t.Errorf("Expected to delete old image %s, got %s", oldImageURL, imageURL)
			}
			deleteCalled = true
			return nil
		},
		uploadFunc: func(file *multipart.FileHeader, metadata map[string]string) (string, error) {
			return "https://imagedelivery.net/test/new-image-id/public", nil
		},
	}

	service := NewAuthorService(mockRepo, mockCloudflare)
	fileHeader := createTestFileHeader("test.jpg", "fake image content")

	_, err := service.UploadImage(1, fileHeader)
	if err != nil {
		t.Errorf("UploadImage() error = %v, want nil", err)
	}

	if !deleteCalled {
		t.Error("Expected old image to be deleted")
	}
}

func TestAuthorService_UploadImage_AuthorNotFound(t *testing.T) {
	mockRepo := &mockAuthorRepository{
		getByIDFunc: func(id uint) (*author.Author, error) {
			return nil, errors.New("record not found")
		},
	}

	mockCloudflare := &mockCloudflareService{}

	service := NewAuthorService(mockRepo, mockCloudflare)
	fileHeader := createTestFileHeader("test.jpg", "fake image content")

	_, err := service.UploadImage(999, fileHeader)
	if err == nil {
		t.Error("Expected error for non-existent author, got nil")
	}
}

func TestAuthorService_UploadImage_CloudflareUploadFails(t *testing.T) {
	mockRepo := &mockAuthorRepository{
		getByIDFunc: func(id uint) (*author.Author, error) {
			return &author.Author{ID: id, Name: "Test Author"}, nil
		},
	}

	mockCloudflare := &mockCloudflareService{
		uploadFunc: func(file *multipart.FileHeader, metadata map[string]string) (string, error) {
			return "", errors.New("cloudflare API error")
		},
	}

	service := NewAuthorService(mockRepo, mockCloudflare)
	fileHeader := createTestFileHeader("test.jpg", "fake image content")

	_, err := service.UploadImage(1, fileHeader)
	if err == nil {
		t.Error("Expected error when Cloudflare upload fails, got nil")
	}
	if err.Error() != "failed to upload image: cloudflare API error" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestAuthorService_UploadImage_DatabaseUpdateFails_Rollback(t *testing.T) {
	uploadedImageURL := "https://imagedelivery.net/test/new-image-id/public"
	rollbackCalled := false

	mockRepo := &mockAuthorRepository{
		getByIDFunc: func(id uint) (*author.Author, error) {
			return &author.Author{ID: id, Name: "Test Author"}, nil
		},
		updateFunc: func(auth *author.Author) error {
			return errors.New("database error")
		},
	}

	mockCloudflare := &mockCloudflareService{
		uploadFunc: func(file *multipart.FileHeader, metadata map[string]string) (string, error) {
			return uploadedImageURL, nil
		},
		deleteFunc: func(imageURL string) error {
			if imageURL != uploadedImageURL {
				t.Errorf("Expected to rollback uploaded image %s, got %s", uploadedImageURL, imageURL)
			}
			rollbackCalled = true
			return nil
		},
	}

	service := NewAuthorService(mockRepo, mockCloudflare)
	fileHeader := createTestFileHeader("test.jpg", "fake image content")

	_, err := service.UploadImage(1, fileHeader)
	if err == nil {
		t.Error("Expected error when database update fails, got nil")
	}

	if !rollbackCalled {
		t.Error("Expected uploaded image to be rolled back when database update fails")
	}
}

func TestAuthorService_DeleteImage_Success(t *testing.T) {
	existingImageURL := "https://imagedelivery.net/test/image-id/public"
	deleteCloudflareCalledauthor := false

	mockRepo := &mockAuthorRepository{
		getByIDFunc: func(id uint) (*author.Author, error) {
			return &author.Author{
				ID:       id,
				Name:     "Test Author",
				ImageURL: existingImageURL,
			}, nil
		},
		updateFunc: func(auth *author.Author) error {
			if auth.ImageURL != "" {
				t.Error("Expected imageURL to be cleared")
			}
			return nil
		},
	}

	mockCloudflare := &mockCloudflareService{
		deleteFunc: func(imageURL string) error {
			if imageURL != existingImageURL {
				t.Errorf("Expected to delete %s, got %s", existingImageURL, imageURL)
			}
			deleteCloudflareCalledauthor = true
			return nil
		},
	}

	service := NewAuthorService(mockRepo, mockCloudflare)

	err := service.DeleteImage(1)
	if err != nil {
		t.Errorf("DeleteImage() error = %v, want nil", err)
	}

	if !deleteCloudflareCalledauthor {
		t.Error("Expected Cloudflare delete to be called")
	}
}

func TestAuthorService_DeleteImage_NoImage(t *testing.T) {
	mockRepo := &mockAuthorRepository{
		getByIDFunc: func(id uint) (*author.Author, error) {
			return &author.Author{
				ID:       id,
				Name:     "Test Author",
				ImageURL: "",
			}, nil
		},
	}

	mockCloudflare := &mockCloudflareService{}

	service := NewAuthorService(mockRepo, mockCloudflare)

	err := service.DeleteImage(1)
	if err == nil {
		t.Error("Expected error when author has no image, got nil")
	}
	if err.Error() != "author has no image to delete" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestAuthorService_DeleteImage_CloudflareFails(t *testing.T) {
	mockRepo := &mockAuthorRepository{
		getByIDFunc: func(id uint) (*author.Author, error) {
			return &author.Author{
				ID:       id,
				Name:     "Test Author",
				ImageURL: "https://imagedelivery.net/test/image-id/public",
			}, nil
		},
	}

	mockCloudflare := &mockCloudflareService{
		deleteFunc: func(imageURL string) error {
			return errors.New("cloudflare API error")
		},
	}

	service := NewAuthorService(mockRepo, mockCloudflare)

	err := service.DeleteImage(1)
	if err == nil {
		t.Error("Expected error when Cloudflare delete fails, got nil")
	}
}

func TestAuthorService_Delete_WithImage(t *testing.T) {
	existingImageURL := "https://imagedelivery.net/test/image-id/public"
	cloudflareDeleteCalled := false
	dbDeleteCalled := false

	mockRepo := &mockAuthorRepository{
		getByIDFunc: func(id uint) (*author.Author, error) {
			return &author.Author{
				ID:       id,
				Name:     "Test Author",
				ImageURL: existingImageURL,
			}, nil
		},
		isReferencedInReportsFunc: func(id uint) (bool, error) {
			return false, nil
		},
		deleteFunc: func(id uint) error {
			dbDeleteCalled = true
			return nil
		},
	}

	mockCloudflare := &mockCloudflareService{
		deleteFunc: func(imageURL string) error {
			if imageURL != existingImageURL {
				t.Errorf("Expected to delete %s, got %s", existingImageURL, imageURL)
			}
			cloudflareDeleteCalled = true
			return nil
		},
	}

	service := NewAuthorService(mockRepo, mockCloudflare)

	err := service.Delete(1)
	if err != nil {
		t.Errorf("Delete() error = %v, want nil", err)
	}

	if !cloudflareDeleteCalled {
		t.Error("Expected Cloudflare image to be deleted")
	}

	if !dbDeleteCalled {
		t.Error("Expected author to be deleted from database")
	}
}

func TestAuthorService_Delete_CloudflareFailsButContinues(t *testing.T) {
	dbDeleteCalled := false

	mockRepo := &mockAuthorRepository{
		getByIDFunc: func(id uint) (*author.Author, error) {
			return &author.Author{
				ID:       id,
				Name:     "Test Author",
				ImageURL: "https://imagedelivery.net/test/image-id/public",
			}, nil
		},
		isReferencedInReportsFunc: func(id uint) (bool, error) {
			return false, nil
		},
		deleteFunc: func(id uint) error {
			dbDeleteCalled = true
			return nil
		},
	}

	mockCloudflare := &mockCloudflareService{
		deleteFunc: func(imageURL string) error {
			return fmt.Errorf("cloudflare API error")
		},
	}

	service := NewAuthorService(mockRepo, mockCloudflare)

	// Should not fail even if Cloudflare delete fails
	err := service.Delete(1)
	if err != nil {
		t.Errorf("Delete() should not fail when Cloudflare delete fails, got error: %v", err)
	}

	if !dbDeleteCalled {
		t.Error("Expected author to still be deleted from database even if Cloudflare fails")
	}
}
