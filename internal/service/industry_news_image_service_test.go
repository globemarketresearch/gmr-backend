package service

import (
	"errors"
	"mime/multipart"
	"testing"
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/industry_news"
)

// Mock IndustryNewsImageRepository for testing
type mockIndustryNewsImageRepository struct {
	createFunc                      func(image *industry_news.IndustryNewsImage) error
	findByIDFunc                    func(id uint) (*industry_news.IndustryNewsImage, error)
	findByIndustryNewsIDFunc        func(industryNewsID uint) ([]industry_news.IndustryNewsImage, error)
	findActiveByIndustryNewsIDFunc  func(industryNewsID uint) ([]industry_news.IndustryNewsImage, error)
	updateFunc                      func(image *industry_news.IndustryNewsImage) error
	deleteFunc                      func(id uint) error
	countByIndustryNewsIDFunc       func(industryNewsID uint) (int64, error)
	countActiveByIndustryNewsIDFunc func(industryNewsID uint) (int64, error)
}

func (m *mockIndustryNewsImageRepository) Create(image *industry_news.IndustryNewsImage) error {
	if m.createFunc != nil {
		return m.createFunc(image)
	}
	image.ID = 1
	return nil
}

func (m *mockIndustryNewsImageRepository) FindByID(id uint) (*industry_news.IndustryNewsImage, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	userID := uint(5)
	return &industry_news.IndustryNewsImage{
		ID:             id,
		IndustryNewsID: 1,
		ImageURL:       "https://imagedelivery.net/test/image-id/public",
		Title:          "Test Image",
		IsActive:       true,
		UploadedBy:     &userID,
	}, nil
}

func (m *mockIndustryNewsImageRepository) FindByIndustryNewsID(industryNewsID uint) ([]industry_news.IndustryNewsImage, error) {
	if m.findByIndustryNewsIDFunc != nil {
		return m.findByIndustryNewsIDFunc(industryNewsID)
	}
	return []industry_news.IndustryNewsImage{}, nil
}

func (m *mockIndustryNewsImageRepository) FindActiveByIndustryNewsID(industryNewsID uint) ([]industry_news.IndustryNewsImage, error) {
	if m.findActiveByIndustryNewsIDFunc != nil {
		return m.findActiveByIndustryNewsIDFunc(industryNewsID)
	}
	return []industry_news.IndustryNewsImage{}, nil
}

func (m *mockIndustryNewsImageRepository) Update(image *industry_news.IndustryNewsImage) error {
	if m.updateFunc != nil {
		return m.updateFunc(image)
	}
	return nil
}

func (m *mockIndustryNewsImageRepository) Delete(id uint) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *mockIndustryNewsImageRepository) CountByIndustryNewsID(industryNewsID uint) (int64, error) {
	if m.countByIndustryNewsIDFunc != nil {
		return m.countByIndustryNewsIDFunc(industryNewsID)
	}
	return 0, nil
}

func (m *mockIndustryNewsImageRepository) CountActiveByIndustryNewsID(industryNewsID uint) (int64, error) {
	if m.countActiveByIndustryNewsIDFunc != nil {
		return m.countActiveByIndustryNewsIDFunc(industryNewsID)
	}
	return 0, nil
}

// Mock IndustryNewsRepository (subset needed for image tests, stubs for the rest)
type mockIndustryNewsRepositoryForImages struct {
	getByIDFunc func(id uint) (*industry_news.IndustryNews, error)
}

func (m *mockIndustryNewsRepositoryForImages) Create(n *industry_news.IndustryNews) error { return nil }

func (m *mockIndustryNewsRepositoryForImages) GetAll(query industry_news.GetIndustryNewsQuery) ([]industry_news.IndustryNews, int64, error) {
	return nil, 0, nil
}

func (m *mockIndustryNewsRepositoryForImages) GetByCategorySlug(categorySlug string, page, limit int) ([]industry_news.IndustryNews, int64, error) {
	return nil, 0, nil
}

func (m *mockIndustryNewsRepositoryForImages) GetByID(id uint) (*industry_news.IndustryNews, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &industry_news.IndustryNews{ID: id, Title: "Test Industry News"}, nil
}

func (m *mockIndustryNewsRepositoryForImages) GetBySlug(slug string) (*industry_news.IndustryNews, error) {
	return nil, nil
}

func (m *mockIndustryNewsRepositoryForImages) Update(id uint, updates map[string]interface{}) error {
	return nil
}

func (m *mockIndustryNewsRepositoryForImages) Delete(id uint) error     { return nil }
func (m *mockIndustryNewsRepositoryForImages) SoftDelete(id uint) error { return nil }
func (m *mockIndustryNewsRepositoryForImages) Restore(id uint) error    { return nil }
func (m *mockIndustryNewsRepositoryForImages) SubmitForReview(id uint) error {
	return nil
}
func (m *mockIndustryNewsRepositoryForImages) Publish(id uint) error   { return nil }
func (m *mockIndustryNewsRepositoryForImages) Unpublish(id uint) error { return nil }
func (m *mockIndustryNewsRepositoryForImages) PublishScheduled(now time.Time) error {
	return nil
}
func (m *mockIndustryNewsRepositoryForImages) SchedulePublish(id uint, publishDate time.Time) error {
	return nil
}
func (m *mockIndustryNewsRepositoryForImages) CancelScheduledPublish(id uint) error { return nil }

func TestIndustryNewsImageService_UploadImage_Success(t *testing.T) {
	uploadedImageURL := "https://imagedelivery.net/test/new-image-id/public"
	userID := uint(5)

	mockImageRepo := &mockIndustryNewsImageRepository{
		createFunc: func(image *industry_news.IndustryNewsImage) error {
			if image.IndustryNewsID != 1 {
				t.Errorf("Expected industryNewsID 1, got %d", image.IndustryNewsID)
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

	mockNewsRepo := &mockIndustryNewsRepositoryForImages{
		getByIDFunc: func(id uint) (*industry_news.IndustryNews, error) {
			return &industry_news.IndustryNews{ID: id, Title: "Test Industry News"}, nil
		},
	}

	mockCloudflare := &mockCloudflareService{
		uploadFunc: func(file *multipart.FileHeader, metadata map[string]string) (string, error) {
			if metadata["industry_news_id"] != "1" {
				t.Errorf("Expected industry_news_id metadata '1', got '%s'", metadata["industry_news_id"])
			}
			if metadata["type"] != "industry_news_image" {
				t.Errorf("Expected type metadata 'industry_news_image', got '%s'", metadata["type"])
			}
			if metadata["uploaded_by"] != "5" {
				t.Errorf("Expected uploaded_by metadata '5', got '%s'", metadata["uploaded_by"])
			}
			return uploadedImageURL, nil
		},
	}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)
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

func TestIndustryNewsImageService_UploadImage_WithoutTitle(t *testing.T) {
	uploadedImageURL := "https://imagedelivery.net/test/new-image-id/public"
	userID := uint(5)

	mockImageRepo := &mockIndustryNewsImageRepository{
		createFunc: func(image *industry_news.IndustryNewsImage) error {
			if image.Title != "" {
				t.Errorf("Expected empty title, got '%s'", image.Title)
			}
			image.ID = 1
			return nil
		},
	}

	mockNewsRepo := &mockIndustryNewsRepositoryForImages{}
	mockCloudflare := &mockCloudflareService{
		uploadFunc: func(file *multipart.FileHeader, metadata map[string]string) (string, error) {
			return uploadedImageURL, nil
		},
	}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)
	fileHeader := createTestFileHeader("chart.png", "fake image content")

	image, err := service.UploadImage(1, fileHeader, "", userID)
	if err != nil {
		t.Errorf("UploadImage() error = %v, want nil", err)
	}

	if image.Title != "" {
		t.Errorf("Expected empty title, got '%s'", image.Title)
	}
}

func TestIndustryNewsImageService_UploadImage_IndustryNewsNotFound(t *testing.T) {
	mockImageRepo := &mockIndustryNewsImageRepository{}
	mockNewsRepo := &mockIndustryNewsRepositoryForImages{
		getByIDFunc: func(id uint) (*industry_news.IndustryNews, error) {
			return nil, errors.New("record not found")
		},
	}
	mockCloudflare := &mockCloudflareService{}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)
	fileHeader := createTestFileHeader("chart.png", "fake image content")

	_, err := service.UploadImage(999, fileHeader, "Test Chart", 5)
	if err == nil {
		t.Error("Expected error for non-existent industry news, got nil")
	}
}

func TestIndustryNewsImageService_UploadImage_CloudflareUploadFails(t *testing.T) {
	mockImageRepo := &mockIndustryNewsImageRepository{}
	mockNewsRepo := &mockIndustryNewsRepositoryForImages{
		getByIDFunc: func(id uint) (*industry_news.IndustryNews, error) {
			return &industry_news.IndustryNews{ID: id, Title: "Test Industry News"}, nil
		},
	}
	mockCloudflare := &mockCloudflareService{
		uploadFunc: func(file *multipart.FileHeader, metadata map[string]string) (string, error) {
			return "", errors.New("cloudflare API error")
		},
	}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)
	fileHeader := createTestFileHeader("chart.png", "fake image content")

	_, err := service.UploadImage(1, fileHeader, "Test Chart", 5)
	if err == nil {
		t.Error("Expected error when Cloudflare upload fails, got nil")
	}
}

func TestIndustryNewsImageService_UploadImage_DatabaseCreateFails_Rollback(t *testing.T) {
	uploadedImageURL := "https://imagedelivery.net/test/new-image-id/public"
	rollbackCalled := false

	mockImageRepo := &mockIndustryNewsImageRepository{
		createFunc: func(image *industry_news.IndustryNewsImage) error {
			return errors.New("database error")
		},
	}

	mockNewsRepo := &mockIndustryNewsRepositoryForImages{
		getByIDFunc: func(id uint) (*industry_news.IndustryNews, error) {
			return &industry_news.IndustryNews{ID: id, Title: "Test Industry News"}, nil
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

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)
	fileHeader := createTestFileHeader("chart.png", "fake image content")

	_, err := service.UploadImage(1, fileHeader, "Test Chart", 5)
	if err == nil {
		t.Error("Expected error when database create fails, got nil")
	}

	if !rollbackCalled {
		t.Error("Expected uploaded image to be rolled back when database create fails")
	}
}

func TestIndustryNewsImageService_UpdateImageMetadata_Success(t *testing.T) {
	newTitle := "Updated Title"
	isActive := false

	mockImageRepo := &mockIndustryNewsImageRepository{
		findByIDFunc: func(id uint) (*industry_news.IndustryNewsImage, error) {
			userID := uint(5)
			return &industry_news.IndustryNewsImage{
				ID:             id,
				IndustryNewsID: 1,
				ImageURL:       "https://imagedelivery.net/test/image-id/public",
				Title:          "Original Title",
				IsActive:       true,
				UploadedBy:     &userID,
			}, nil
		},
		updateFunc: func(image *industry_news.IndustryNewsImage) error {
			if image.Title != newTitle {
				t.Errorf("Expected title '%s', got '%s'", newTitle, image.Title)
			}
			if image.IsActive != isActive {
				t.Errorf("Expected is_active %v, got %v", isActive, image.IsActive)
			}
			return nil
		},
	}

	mockNewsRepo := &mockIndustryNewsRepositoryForImages{}
	mockCloudflare := &mockCloudflareService{}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)

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

func TestIndustryNewsImageService_UpdateImageMetadata_PartialUpdate_TitleOnly(t *testing.T) {
	newTitle := "Updated Title"

	mockImageRepo := &mockIndustryNewsImageRepository{
		findByIDFunc: func(id uint) (*industry_news.IndustryNewsImage, error) {
			userID := uint(5)
			return &industry_news.IndustryNewsImage{
				ID:             id,
				IndustryNewsID: 1,
				ImageURL:       "https://imagedelivery.net/test/image-id/public",
				Title:          "Original Title",
				IsActive:       true,
				UploadedBy:     &userID,
			}, nil
		},
		updateFunc: func(image *industry_news.IndustryNewsImage) error {
			if image.Title != newTitle {
				t.Errorf("Expected title '%s', got '%s'", newTitle, image.Title)
			}
			if !image.IsActive {
				t.Error("Expected is_active to remain true")
			}
			return nil
		},
	}

	mockNewsRepo := &mockIndustryNewsRepositoryForImages{}
	mockCloudflare := &mockCloudflareService{}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)

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

func TestIndustryNewsImageService_UpdateImageMetadata_ImageNotFound(t *testing.T) {
	mockImageRepo := &mockIndustryNewsImageRepository{
		findByIDFunc: func(id uint) (*industry_news.IndustryNewsImage, error) {
			return nil, errors.New("record not found")
		},
	}

	mockNewsRepo := &mockIndustryNewsRepositoryForImages{}
	mockCloudflare := &mockCloudflareService{}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)

	newTitle := "Updated Title"
	_, err := service.UpdateImageMetadata(999, &newTitle, nil)
	if err == nil {
		t.Error("Expected error for non-existent image, got nil")
	}
}

func TestIndustryNewsImageService_DeleteImage_Success(t *testing.T) {
	deleteCalled := false

	mockImageRepo := &mockIndustryNewsImageRepository{
		findByIDFunc: func(id uint) (*industry_news.IndustryNewsImage, error) {
			userID := uint(5)
			return &industry_news.IndustryNewsImage{
				ID:             id,
				IndustryNewsID: 1,
				ImageURL:       "https://imagedelivery.net/test/image-id/public",
				Title:          "Test Image",
				IsActive:       true,
				UploadedBy:     &userID,
			}, nil
		},
		deleteFunc: func(id uint) error {
			deleteCalled = true
			return nil
		},
	}

	mockNewsRepo := &mockIndustryNewsRepositoryForImages{}
	mockCloudflare := &mockCloudflareService{}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)

	err := service.DeleteImage(1)
	if err != nil {
		t.Errorf("DeleteImage() error = %v, want nil", err)
	}

	if !deleteCalled {
		t.Error("Expected delete to be called")
	}
}

func TestIndustryNewsImageService_DeleteImage_CloudflareErrorStillDeletes(t *testing.T) {
	deleteCalled := false

	mockImageRepo := &mockIndustryNewsImageRepository{
		findByIDFunc: func(id uint) (*industry_news.IndustryNewsImage, error) {
			userID := uint(5)
			return &industry_news.IndustryNewsImage{
				ID:             id,
				IndustryNewsID: 1,
				ImageURL:       "https://imagedelivery.net/test/image-id/public",
				Title:          "Test Image",
				IsActive:       true,
				UploadedBy:     &userID,
			}, nil
		},
		deleteFunc: func(id uint) error {
			deleteCalled = true
			return nil
		},
	}

	mockNewsRepo := &mockIndustryNewsRepositoryForImages{}
	mockCloudflare := &mockCloudflareService{
		deleteFunc: func(imageURL string) error {
			return errors.New("cloudflare delete error")
		},
	}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)

	err := service.DeleteImage(1)
	if err != nil {
		t.Errorf("DeleteImage() error = %v, want nil", err)
	}

	if !deleteCalled {
		t.Error("Expected DB delete to be called even when Cloudflare delete fails")
	}
}

func TestIndustryNewsImageService_DeleteImage_ImageNotFound(t *testing.T) {
	mockImageRepo := &mockIndustryNewsImageRepository{
		findByIDFunc: func(id uint) (*industry_news.IndustryNewsImage, error) {
			return nil, errors.New("record not found")
		},
	}

	mockNewsRepo := &mockIndustryNewsRepositoryForImages{}
	mockCloudflare := &mockCloudflareService{}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)

	err := service.DeleteImage(999)
	if err == nil {
		t.Error("Expected error for non-existent image, got nil")
	}
}

func TestIndustryNewsImageService_GetImagesByIndustryNews_AllImages(t *testing.T) {
	userID := uint(5)
	expectedImages := []industry_news.IndustryNewsImage{
		{ID: 1, IndustryNewsID: 1, ImageURL: "url1", Title: "Image 1", IsActive: true, UploadedBy: &userID},
		{ID: 2, IndustryNewsID: 1, ImageURL: "url2", Title: "Image 2", IsActive: false, UploadedBy: &userID},
		{ID: 3, IndustryNewsID: 1, ImageURL: "url3", Title: "Image 3", IsActive: true, UploadedBy: &userID},
	}

	mockImageRepo := &mockIndustryNewsImageRepository{
		findByIndustryNewsIDFunc: func(industryNewsID uint) ([]industry_news.IndustryNewsImage, error) {
			return expectedImages, nil
		},
	}

	mockNewsRepo := &mockIndustryNewsRepositoryForImages{}
	mockCloudflare := &mockCloudflareService{}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)

	images, err := service.GetImagesByIndustryNews(1, false)
	if err != nil {
		t.Errorf("GetImagesByIndustryNews() error = %v, want nil", err)
	}

	if len(images) != 3 {
		t.Errorf("Expected 3 images, got %d", len(images))
	}
}

func TestIndustryNewsImageService_GetImagesByIndustryNews_ActiveOnly(t *testing.T) {
	userID := uint(5)
	expectedImages := []industry_news.IndustryNewsImage{
		{ID: 1, IndustryNewsID: 1, ImageURL: "url1", Title: "Image 1", IsActive: true, UploadedBy: &userID},
		{ID: 3, IndustryNewsID: 1, ImageURL: "url3", Title: "Image 3", IsActive: true, UploadedBy: &userID},
	}

	mockImageRepo := &mockIndustryNewsImageRepository{
		findActiveByIndustryNewsIDFunc: func(industryNewsID uint) ([]industry_news.IndustryNewsImage, error) {
			return expectedImages, nil
		},
	}

	mockNewsRepo := &mockIndustryNewsRepositoryForImages{}
	mockCloudflare := &mockCloudflareService{}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)

	images, err := service.GetImagesByIndustryNews(1, true)
	if err != nil {
		t.Errorf("GetImagesByIndustryNews() error = %v, want nil", err)
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

func TestIndustryNewsImageService_GetImageByID_Success(t *testing.T) {
	userID := uint(5)
	mockImageRepo := &mockIndustryNewsImageRepository{
		findByIDFunc: func(id uint) (*industry_news.IndustryNewsImage, error) {
			return &industry_news.IndustryNewsImage{
				ID:             id,
				IndustryNewsID: 1,
				ImageURL:       "https://imagedelivery.net/test/image-id/public",
				Title:          "Test Image",
				IsActive:       true,
				UploadedBy:     &userID,
			}, nil
		},
	}

	mockNewsRepo := &mockIndustryNewsRepositoryForImages{}
	mockCloudflare := &mockCloudflareService{}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)

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

func TestIndustryNewsImageService_GetImageByID_NotFound(t *testing.T) {
	mockImageRepo := &mockIndustryNewsImageRepository{
		findByIDFunc: func(id uint) (*industry_news.IndustryNewsImage, error) {
			return nil, errors.New("record not found")
		},
	}

	mockNewsRepo := &mockIndustryNewsRepositoryForImages{}
	mockCloudflare := &mockCloudflareService{}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)

	_, err := service.GetImageByID(999)
	if err == nil {
		t.Error("Expected error for non-existent image, got nil")
	}
}
