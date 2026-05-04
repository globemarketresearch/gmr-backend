package service

import (
	"errors"
	"testing"

	"github.com/healthcare-market-research/backend/internal/domain/report"
)

// Mock ReportImageRepository for testing
type mockReportImageRepository struct {
	createFunc                func(image *report.ReportImage) error
	findByIDFunc              func(id uint) (*report.ReportImage, error)
	findByReportIDFunc        func(reportID uint) ([]report.ReportImage, error)
	findActiveByReportIDFunc  func(reportID uint) ([]report.ReportImage, error)
	updateFunc                func(image *report.ReportImage) error
	deleteFunc                func(id uint) error
	countByReportIDFunc       func(reportID uint) (int64, error)
	countActiveByReportIDFunc func(reportID uint) (int64, error)
}

func (m *mockReportImageRepository) Create(image *report.ReportImage) error {
	if m.createFunc != nil {
		return m.createFunc(image)
	}
	image.ID = 1
	return nil
}

func (m *mockReportImageRepository) FindByID(id uint) (*report.ReportImage, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	userID := uint(5)
	return &report.ReportImage{
		ID:         id,
		ReportID:   1,
		ImageURL:   "https://imagedelivery.net/test/image-id/public",
		Title:      "Test Image",
		IsActive:   true,
		UploadedBy: &userID,
	}, nil
}

func (m *mockReportImageRepository) FindByReportID(reportID uint) ([]report.ReportImage, error) {
	if m.findByReportIDFunc != nil {
		return m.findByReportIDFunc(reportID)
	}
	return []report.ReportImage{}, nil
}

func (m *mockReportImageRepository) FindActiveByReportID(reportID uint) ([]report.ReportImage, error) {
	if m.findActiveByReportIDFunc != nil {
		return m.findActiveByReportIDFunc(reportID)
	}
	return []report.ReportImage{}, nil
}

func (m *mockReportImageRepository) Update(image *report.ReportImage) error {
	if m.updateFunc != nil {
		return m.updateFunc(image)
	}
	return nil
}

func (m *mockReportImageRepository) Delete(id uint) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *mockReportImageRepository) CountByReportID(reportID uint) (int64, error) {
	if m.countByReportIDFunc != nil {
		return m.countByReportIDFunc(reportID)
	}
	return 0, nil
}

func (m *mockReportImageRepository) CountActiveByReportID(reportID uint) (int64, error) {
	if m.countActiveByReportIDFunc != nil {
		return m.countActiveByReportIDFunc(reportID)
	}
	return 0, nil
}

// Mock ReportRepository for testing
type mockReportRepository struct {
	getByIDFunc func(id uint) (*report.Report, error)
}

func (m *mockReportRepository) GetByID(id uint) (*report.Report, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &report.Report{ID: id, Title: "Test Report"}, nil
}

func (m *mockReportRepository) GetAll(page, limit int) ([]report.Report, int64, error) {
	return nil, 0, nil
}

func (m *mockReportRepository) GetAllWithFilters(filters interface{}) ([]report.Report, int64, error) {
	return nil, 0, nil
}

func (m *mockReportRepository) GetBySlug(slug string) (*report.ReportWithRelations, error) {
	return nil, nil
}

func (m *mockReportRepository) GetByIDWithRelations(id uint) (*report.ReportWithRelations, error) {
	return nil, nil
}

func (m *mockReportRepository) GetByCategorySlug(categorySlug string, page, limit int) ([]report.Report, int64, error) {
	return nil, 0, nil
}

func (m *mockReportRepository) Search(query string, page, limit int) ([]report.Report, int64, error) {
	return nil, 0, nil
}

func (m *mockReportRepository) GetChartsByReportID(reportID uint) ([]report.ChartMetadata, error) {
	return nil, nil
}

func (m *mockReportRepository) Create(report *report.Report) error {
	return nil
}

func (m *mockReportRepository) Update(report *report.Report) error {
	return nil
}

func (m *mockReportRepository) Delete(id uint) error {
	return nil
}

func TestReportImageService_UploadImage_Success(t *testing.T) {
	uploadedImageURL := "https://imagedelivery.net/test/new-image-id/public"
	userID := uint(5)

	mockReportImageRepo := &mockReportImageRepository{
		createFunc: func(image *report.ReportImage) error {
			if image.ReportID != 1 {
				t.Errorf("Expected reportID 1, got %d", image.ReportID)
			}
			if image.ImageURL != uploadedImageURL {
				t.Errorf("Expected imageURL %s, got %s", uploadedImageURL, image.ImageURL)
			}
			if image.Title != "Test Chart" {
				t.Errorf("Expected title 'Test Chart', got %s", image.Title)
			}
			if !image.IsActive {
				t.Error("Expected is_active to be true")
			}
			if *image.UploadedBy != userID {
				t.Errorf("Expected uploaded_by %d, got %d", userID, *image.UploadedBy)
			}
			image.ID = 1
			return nil
		},
	}

	mockReportRepo := &mockReportRepository{
		getByIDFunc: func(id uint) (*report.Report, error) {
			return &report.Report{ID: id, Title: "Test Report"}, nil
		},
	}

	mockCloudflare := &mockCloudflareService{
		uploadFunc: func(file interface{}, metadata map[string]string) (string, error) {
			if metadata["report_id"] != "1" {
				t.Errorf("Expected report_id metadata '1', got '%s'", metadata["report_id"])
			}
			if metadata["type"] != "report_image" {
				t.Errorf("Expected type metadata 'report_image', got '%s'", metadata["type"])
			}
			if metadata["uploaded_by"] != "5" {
				t.Errorf("Expected uploaded_by metadata '5', got '%s'", metadata["uploaded_by"])
			}
			return uploadedImageURL, nil
		},
	}

	service := NewReportImageService(mockReportImageRepo, mockReportRepo, mockCloudflare)
	fileHeader := createTestFileHeader("chart.png", "fake image content")

	image, err := service.UploadImage(1, fileHeader, "Test Chart", userID)
	if err != nil {
		t.Errorf("UploadImage() error = %v, want nil", err)
	}

	if image.ID != 1 {
		t.Errorf("Expected image ID 1, got %d", image.ID)
	}
	if image.ImageURL != uploadedImageURL {
		t.Errorf("Expected imageURL %s, got %s", uploadedImageURL, image.ImageURL)
	}
}

func TestReportImageService_UploadImage_WithoutTitle(t *testing.T) {
	uploadedImageURL := "https://imagedelivery.net/test/new-image-id/public"
	userID := uint(5)

	mockReportImageRepo := &mockReportImageRepository{
		createFunc: func(image *report.ReportImage) error {
			if image.Title != "" {
				t.Errorf("Expected empty title, got '%s'", image.Title)
			}
			image.ID = 1
			return nil
		},
	}

	mockReportRepo := &mockReportRepository{}
	mockCloudflare := &mockCloudflareService{
		uploadFunc: func(file interface{}, metadata map[string]string) (string, error) {
			return uploadedImageURL, nil
		},
	}

	service := NewReportImageService(mockReportImageRepo, mockReportRepo, mockCloudflare)
	fileHeader := createTestFileHeader("chart.png", "fake image content")

	image, err := service.UploadImage(1, fileHeader, "", userID)
	if err != nil {
		t.Errorf("UploadImage() error = %v, want nil", err)
	}

	if image.Title != "" {
		t.Errorf("Expected empty title, got '%s'", image.Title)
	}
}

func TestReportImageService_UploadImage_ReportNotFound(t *testing.T) {
	mockReportImageRepo := &mockReportImageRepository{}
	mockReportRepo := &mockReportRepository{
		getByIDFunc: func(id uint) (*report.Report, error) {
			return nil, errors.New("record not found")
		},
	}
	mockCloudflare := &mockCloudflareService{}

	service := NewReportImageService(mockReportImageRepo, mockReportRepo, mockCloudflare)
	fileHeader := createTestFileHeader("chart.png", "fake image content")

	_, err := service.UploadImage(999, fileHeader, "Test Chart", 5)
	if err == nil {
		t.Error("Expected error for non-existent report, got nil")
	}
}

func TestReportImageService_UploadImage_CloudflareUploadFails(t *testing.T) {
	mockReportImageRepo := &mockReportImageRepository{}
	mockReportRepo := &mockReportRepository{
		getByIDFunc: func(id uint) (*report.Report, error) {
			return &report.Report{ID: id, Title: "Test Report"}, nil
		},
	}
	mockCloudflare := &mockCloudflareService{
		uploadFunc: func(file interface{}, metadata map[string]string) (string, error) {
			return "", errors.New("cloudflare API error")
		},
	}

	service := NewReportImageService(mockReportImageRepo, mockReportRepo, mockCloudflare)
	fileHeader := createTestFileHeader("chart.png", "fake image content")

	_, err := service.UploadImage(1, fileHeader, "Test Chart", 5)
	if err == nil {
		t.Error("Expected error when Cloudflare upload fails, got nil")
	}
}

func TestReportImageService_UploadImage_DatabaseCreateFails_Rollback(t *testing.T) {
	uploadedImageURL := "https://imagedelivery.net/test/new-image-id/public"
	rollbackCalled := false

	mockReportImageRepo := &mockReportImageRepository{
		createFunc: func(image *report.ReportImage) error {
			return errors.New("database error")
		},
	}

	mockReportRepo := &mockReportRepository{
		getByIDFunc: func(id uint) (*report.Report, error) {
			return &report.Report{ID: id, Title: "Test Report"}, nil
		},
	}

	mockCloudflare := &mockCloudflareService{
		uploadFunc: func(file interface{}, metadata map[string]string) (string, error) {
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

	service := NewReportImageService(mockReportImageRepo, mockReportRepo, mockCloudflare)
	fileHeader := createTestFileHeader("chart.png", "fake image content")

	_, err := service.UploadImage(1, fileHeader, "Test Chart", 5)
	if err == nil {
		t.Error("Expected error when database create fails, got nil")
	}

	if !rollbackCalled {
		t.Error("Expected uploaded image to be rolled back when database create fails")
	}
}

func TestReportImageService_UpdateImageMetadata_Success(t *testing.T) {
	newTitle := "Updated Title"
	isActive := false

	mockReportImageRepo := &mockReportImageRepository{
		findByIDFunc: func(id uint) (*report.ReportImage, error) {
			userID := uint(5)
			return &report.ReportImage{
				ID:         id,
				ReportID:   1,
				ImageURL:   "https://imagedelivery.net/test/image-id/public",
				Title:      "Original Title",
				IsActive:   true,
				UploadedBy: &userID,
			}, nil
		},
		updateFunc: func(image *report.ReportImage) error {
			if image.Title != newTitle {
				t.Errorf("Expected title '%s', got '%s'", newTitle, image.Title)
			}
			if image.IsActive != isActive {
				t.Errorf("Expected is_active %v, got %v", isActive, image.IsActive)
			}
			return nil
		},
	}

	mockReportRepo := &mockReportRepository{}
	mockCloudflare := &mockCloudflareService{}

	service := NewReportImageService(mockReportImageRepo, mockReportRepo, mockCloudflare)

	updatedImage, err := service.UpdateImageMetadata(1, &newTitle, &isActive)
	if err != nil {
		t.Errorf("UpdateImageMetadata() error = %v, want nil", err)
	}

	if updatedImage.Title != newTitle {
		t.Errorf("Expected title '%s', got '%s'", newTitle, updatedImage.Title)
	}
	if updatedImage.IsActive != isActive {
		t.Errorf("Expected is_active %v, got %v", isActive, updatedImage.IsActive)
	}
}

func TestReportImageService_UpdateImageMetadata_PartialUpdate_TitleOnly(t *testing.T) {
	newTitle := "Updated Title"

	mockReportImageRepo := &mockReportImageRepository{
		findByIDFunc: func(id uint) (*report.ReportImage, error) {
			userID := uint(5)
			return &report.ReportImage{
				ID:         id,
				ReportID:   1,
				ImageURL:   "https://imagedelivery.net/test/image-id/public",
				Title:      "Original Title",
				IsActive:   true,
				UploadedBy: &userID,
			}, nil
		},
		updateFunc: func(image *report.ReportImage) error {
			if image.Title != newTitle {
				t.Errorf("Expected title '%s', got '%s'", newTitle, image.Title)
			}
			if !image.IsActive {
				t.Error("Expected is_active to remain true")
			}
			return nil
		},
	}

	mockReportRepo := &mockReportRepository{}
	mockCloudflare := &mockCloudflareService{}

	service := NewReportImageService(mockReportImageRepo, mockReportRepo, mockCloudflare)

	updatedImage, err := service.UpdateImageMetadata(1, &newTitle, nil)
	if err != nil {
		t.Errorf("UpdateImageMetadata() error = %v, want nil", err)
	}

	if updatedImage.Title != newTitle {
		t.Errorf("Expected title '%s', got '%s'", newTitle, updatedImage.Title)
	}
	if !updatedImage.IsActive {
		t.Error("Expected is_active to remain true")
	}
}

func TestReportImageService_UpdateImageMetadata_ImageNotFound(t *testing.T) {
	mockReportImageRepo := &mockReportImageRepository{
		findByIDFunc: func(id uint) (*report.ReportImage, error) {
			return nil, errors.New("record not found")
		},
	}

	mockReportRepo := &mockReportRepository{}
	mockCloudflare := &mockCloudflareService{}

	service := NewReportImageService(mockReportImageRepo, mockReportRepo, mockCloudflare)

	newTitle := "Updated Title"
	_, err := service.UpdateImageMetadata(999, &newTitle, nil)
	if err == nil {
		t.Error("Expected error for non-existent image, got nil")
	}
}

func TestReportImageService_DeleteImage_Success(t *testing.T) {
	deleteCalled := false

	mockReportImageRepo := &mockReportImageRepository{
		findByIDFunc: func(id uint) (*report.ReportImage, error) {
			userID := uint(5)
			return &report.ReportImage{
				ID:         id,
				ReportID:   1,
				ImageURL:   "https://imagedelivery.net/test/image-id/public",
				Title:      "Test Image",
				IsActive:   true,
				UploadedBy: &userID,
			}, nil
		},
		deleteFunc: func(id uint) error {
			deleteCalled = true
			return nil
		},
	}

	mockReportRepo := &mockReportRepository{}
	mockCloudflare := &mockCloudflareService{}

	service := NewReportImageService(mockReportImageRepo, mockReportRepo, mockCloudflare)

	err := service.DeleteImage(1)
	if err != nil {
		t.Errorf("DeleteImage() error = %v, want nil", err)
	}

	if !deleteCalled {
		t.Error("Expected delete to be called")
	}
}

func TestReportImageService_DeleteImage_ImageNotFound(t *testing.T) {
	mockReportImageRepo := &mockReportImageRepository{
		findByIDFunc: func(id uint) (*report.ReportImage, error) {
			return nil, errors.New("record not found")
		},
	}

	mockReportRepo := &mockReportRepository{}
	mockCloudflare := &mockCloudflareService{}

	service := NewReportImageService(mockReportImageRepo, mockReportRepo, mockCloudflare)

	err := service.DeleteImage(999)
	if err == nil {
		t.Error("Expected error for non-existent image, got nil")
	}
}

func TestReportImageService_GetImagesByReport_AllImages(t *testing.T) {
	userID := uint(5)
	expectedImages := []report.ReportImage{
		{ID: 1, ReportID: 1, ImageURL: "url1", Title: "Image 1", IsActive: true, UploadedBy: &userID},
		{ID: 2, ReportID: 1, ImageURL: "url2", Title: "Image 2", IsActive: false, UploadedBy: &userID},
		{ID: 3, ReportID: 1, ImageURL: "url3", Title: "Image 3", IsActive: true, UploadedBy: &userID},
	}

	mockReportImageRepo := &mockReportImageRepository{
		findByReportIDFunc: func(reportID uint) ([]report.ReportImage, error) {
			return expectedImages, nil
		},
	}

	mockReportRepo := &mockReportRepository{}
	mockCloudflare := &mockCloudflareService{}

	service := NewReportImageService(mockReportImageRepo, mockReportRepo, mockCloudflare)

	images, err := service.GetImagesByReport(1, false)
	if err != nil {
		t.Errorf("GetImagesByReport() error = %v, want nil", err)
	}

	if len(images) != 3 {
		t.Errorf("Expected 3 images, got %d", len(images))
	}
}

func TestReportImageService_GetImagesByReport_ActiveOnly(t *testing.T) {
	userID := uint(5)
	expectedImages := []report.ReportImage{
		{ID: 1, ReportID: 1, ImageURL: "url1", Title: "Image 1", IsActive: true, UploadedBy: &userID},
		{ID: 3, ReportID: 1, ImageURL: "url3", Title: "Image 3", IsActive: true, UploadedBy: &userID},
	}

	mockReportImageRepo := &mockReportImageRepository{
		findActiveByReportIDFunc: func(reportID uint) ([]report.ReportImage, error) {
			return expectedImages, nil
		},
	}

	mockReportRepo := &mockReportRepository{}
	mockCloudflare := &mockCloudflareService{}

	service := NewReportImageService(mockReportImageRepo, mockReportRepo, mockCloudflare)

	images, err := service.GetImagesByReport(1, true)
	if err != nil {
		t.Errorf("GetImagesByReport() error = %v, want nil", err)
	}

	if len(images) != 2 {
		t.Errorf("Expected 2 active images, got %d", len(images))
	}

	for _, img := range images {
		if !img.IsActive {
			t.Error("Expected all images to be active")
		}
	}
}

func TestReportImageService_GetImageByID_Success(t *testing.T) {
	userID := uint(5)
	mockReportImageRepo := &mockReportImageRepository{
		findByIDFunc: func(id uint) (*report.ReportImage, error) {
			return &report.ReportImage{
				ID:         id,
				ReportID:   1,
				ImageURL:   "https://imagedelivery.net/test/image-id/public",
				Title:      "Test Image",
				IsActive:   true,
				UploadedBy: &userID,
			}, nil
		},
	}

	mockReportRepo := &mockReportRepository{}
	mockCloudflare := &mockCloudflareService{}

	service := NewReportImageService(mockReportImageRepo, mockReportRepo, mockCloudflare)

	image, err := service.GetImageByID(1)
	if err != nil {
		t.Errorf("GetImageByID() error = %v, want nil", err)
	}

	if image.ID != 1 {
		t.Errorf("Expected image ID 1, got %d", image.ID)
	}
	if image.Title != "Test Image" {
		t.Errorf("Expected title 'Test Image', got '%s'", image.Title)
	}
}

func TestReportImageService_GetImageByID_NotFound(t *testing.T) {
	mockReportImageRepo := &mockReportImageRepository{
		findByIDFunc: func(id uint) (*report.ReportImage, error) {
			return nil, errors.New("record not found")
		},
	}

	mockReportRepo := &mockReportRepository{}
	mockCloudflare := &mockCloudflareService{}

	service := NewReportImageService(mockReportImageRepo, mockReportRepo, mockCloudflare)

	_, err := service.GetImageByID(999)
	if err == nil {
		t.Error("Expected error for non-existent image, got nil")
	}
}
