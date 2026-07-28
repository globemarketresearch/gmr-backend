package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/healthcare-market-research/backend/internal/domain/category"
	"github.com/healthcare-market-research/backend/internal/domain/industry_news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupIndustryNewsImageTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&category.Category{}, &industry_news.IndustryNews{}, &industry_news.IndustryNewsImage{})
	require.NoError(t, err)

	return db
}

func createTestIndustryNews(t *testing.T, db *gorm.DB) uint {
	suffix := uuid.New().String()
	cat := &category.Category{Name: "Test Category", Slug: "test-category-" + suffix}
	require.NoError(t, db.Create(cat).Error)

	item := &industry_news.IndustryNews{
		Title:      "A Sufficiently Long Test Title",
		Slug:       "a-sufficiently-long-test-title-" + suffix,
		Excerpt:    "An excerpt that is long enough to satisfy the fifty character minimum requirement.",
		Content:    "Content that is long enough to satisfy the one hundred character minimum requirement for industry news articles in this test suite.",
		CategoryID: cat.ID,
		AuthorID:   1,
		Status:     industry_news.StatusDraft,
	}
	require.NoError(t, db.Create(item).Error)
	return item.ID
}

func TestIndustryNewsImageRepository_Create(t *testing.T) {
	db := setupIndustryNewsImageTestDB(t)
	repo := NewIndustryNewsImageRepository(db)
	newsID := createTestIndustryNews(t, db)

	userID := uint(5)
	image := &industry_news.IndustryNewsImage{
		IndustryNewsID: newsID,
		ImageURL:       "https://example.com/image1.png",
		Title:          "Test Image",
		IsActive:       true,
		UploadedBy:     &userID,
	}

	err := repo.Create(image)
	require.NoError(t, err)
	assert.NotZero(t, image.ID)
	assert.NotZero(t, image.CreatedAt)
	assert.NotZero(t, image.UpdatedAt)
}

func TestIndustryNewsImageRepository_FindByID(t *testing.T) {
	db := setupIndustryNewsImageTestDB(t)
	repo := NewIndustryNewsImageRepository(db)
	newsID := createTestIndustryNews(t, db)

	userID := uint(5)
	image := &industry_news.IndustryNewsImage{
		IndustryNewsID: newsID,
		ImageURL:       "https://example.com/image1.png",
		Title:          "Test Image",
		IsActive:       true,
		UploadedBy:     &userID,
	}

	err := repo.Create(image)
	require.NoError(t, err)

	t.Run("Successfully retrieve image by ID", func(t *testing.T) {
		result, err := repo.FindByID(image.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, image.ID, result.ID)
		assert.Equal(t, image.ImageURL, result.ImageURL)
		assert.Equal(t, image.Title, result.Title)
		assert.Equal(t, true, result.IsActive)
	})

	t.Run("Return error for non-existent ID", func(t *testing.T) {
		result, err := repo.FindByID(999)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestIndustryNewsImageRepository_FindByIndustryNewsID(t *testing.T) {
	db := setupIndustryNewsImageTestDB(t)
	repo := NewIndustryNewsImageRepository(db)
	newsID := createTestIndustryNews(t, db)

	userID := uint(5)
	images := []*industry_news.IndustryNewsImage{
		{
			IndustryNewsID: newsID,
			ImageURL:       "https://example.com/image1.png",
			Title:          "Image 1",
			IsActive:       true,
			UploadedBy:     &userID,
		},
		{
			IndustryNewsID: newsID,
			ImageURL:       "https://example.com/image2.png",
			Title:          "Image 2",
			IsActive:       false,
			UploadedBy:     &userID,
		},
		{
			IndustryNewsID: newsID,
			ImageURL:       "https://example.com/image3.png",
			Title:          "Image 3",
			IsActive:       true,
			UploadedBy:     &userID,
		},
	}

	for _, img := range images {
		err := repo.Create(img)
		require.NoError(t, err)
		// Ensure distinct created_at values so DESC ordering assertions are
		// deterministic even on fast machines / low-resolution clocks.
		time.Sleep(2 * time.Millisecond)
	}

	t.Run("Retrieve all images for industry news", func(t *testing.T) {
		results, err := repo.FindByIndustryNewsID(newsID)
		require.NoError(t, err)
		assert.Len(t, results, 3)
		// Should be ordered by created_at DESC (newest first)
		assert.Equal(t, "Image 3", results[0].Title)
		assert.Equal(t, "Image 2", results[1].Title)
		assert.Equal(t, "Image 1", results[2].Title)
	})

	t.Run("Return empty array for industry news with no images", func(t *testing.T) {
		newNewsID := createTestIndustryNews(t, db)
		results, err := repo.FindByIndustryNewsID(newNewsID)
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})
}

func TestIndustryNewsImageRepository_FindActiveByIndustryNewsID(t *testing.T) {
	db := setupIndustryNewsImageTestDB(t)
	repo := NewIndustryNewsImageRepository(db)
	newsID := createTestIndustryNews(t, db)

	userID := uint(5)
	images := []*industry_news.IndustryNewsImage{
		{
			IndustryNewsID: newsID,
			ImageURL:       "https://example.com/image1.png",
			Title:          "Active Image 1",
			IsActive:       true,
			UploadedBy:     &userID,
		},
		{
			IndustryNewsID: newsID,
			ImageURL:       "https://example.com/image2.png",
			Title:          "Inactive Image",
			IsActive:       false,
			UploadedBy:     &userID,
		},
		{
			IndustryNewsID: newsID,
			ImageURL:       "https://example.com/image3.png",
			Title:          "Active Image 2",
			IsActive:       true,
			UploadedBy:     &userID,
		},
	}

	for _, img := range images {
		err := repo.Create(img)
		require.NoError(t, err)
		// Ensure distinct created_at values so DESC ordering assertions are
		// deterministic even on fast machines / low-resolution clocks.
		time.Sleep(2 * time.Millisecond)
	}

	t.Run("Retrieve only active images", func(t *testing.T) {
		results, err := repo.FindActiveByIndustryNewsID(newsID)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		assert.True(t, results[0].IsActive)
		assert.True(t, results[1].IsActive)
		// Should be ordered by created_at DESC
		assert.Equal(t, "Active Image 2", results[0].Title)
		assert.Equal(t, "Active Image 1", results[1].Title)
	})

	t.Run("Return empty array when no active images", func(t *testing.T) {
		newNewsID := createTestIndustryNews(t, db)
		inactiveImage := &industry_news.IndustryNewsImage{
			IndustryNewsID: newNewsID,
			ImageURL:       "https://example.com/inactive.png",
			Title:          "Inactive",
			IsActive:       false,
			UploadedBy:     &userID,
		}
		err := repo.Create(inactiveImage)
		require.NoError(t, err)

		results, err := repo.FindActiveByIndustryNewsID(newNewsID)
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})
}

func TestIndustryNewsImageRepository_Update(t *testing.T) {
	db := setupIndustryNewsImageTestDB(t)
	repo := NewIndustryNewsImageRepository(db)
	newsID := createTestIndustryNews(t, db)

	userID := uint(5)
	image := &industry_news.IndustryNewsImage{
		IndustryNewsID: newsID,
		ImageURL:       "https://example.com/image1.png",
		Title:          "Original Title",
		IsActive:       true,
		UploadedBy:     &userID,
	}

	err := repo.Create(image)
	require.NoError(t, err)

	// Update the image
	image.Title = "Updated Title"
	image.IsActive = false

	err = repo.Update(image)
	require.NoError(t, err)

	// Verify update
	result, err := repo.FindByID(image.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", result.Title)
	assert.False(t, result.IsActive)
}

func TestIndustryNewsImageRepository_Delete(t *testing.T) {
	db := setupIndustryNewsImageTestDB(t)
	repo := NewIndustryNewsImageRepository(db)
	newsID := createTestIndustryNews(t, db)

	userID := uint(5)
	image := &industry_news.IndustryNewsImage{
		IndustryNewsID: newsID,
		ImageURL:       "https://example.com/image1.png",
		Title:          "Test Image",
		IsActive:       true,
		UploadedBy:     &userID,
	}

	err := repo.Create(image)
	require.NoError(t, err)

	// Hard delete
	err = repo.Delete(image.ID)
	require.NoError(t, err)

	// Verify the image is gone from DB
	_, err = repo.FindByID(image.ID)
	assert.Error(t, err)

	// Verify it doesn't appear in active results
	activeImages, err := repo.FindActiveByIndustryNewsID(newsID)
	require.NoError(t, err)
	assert.Len(t, activeImages, 0)

	// Verify it doesn't appear in all results either
	allImages, err := repo.FindByIndustryNewsID(newsID)
	require.NoError(t, err)
	assert.Len(t, allImages, 0)
}

func TestIndustryNewsImageRepository_CountByIndustryNewsID(t *testing.T) {
	db := setupIndustryNewsImageTestDB(t)
	repo := NewIndustryNewsImageRepository(db)
	newsID := createTestIndustryNews(t, db)

	userID := uint(5)
	images := []*industry_news.IndustryNewsImage{
		{
			IndustryNewsID: newsID,
			ImageURL:       "https://example.com/image1.png",
			Title:          "Image 1",
			IsActive:       true,
			UploadedBy:     &userID,
		},
		{
			IndustryNewsID: newsID,
			ImageURL:       "https://example.com/image2.png",
			Title:          "Image 2",
			IsActive:       false,
			UploadedBy:     &userID,
		},
		{
			IndustryNewsID: newsID,
			ImageURL:       "https://example.com/image3.png",
			Title:          "Image 3",
			IsActive:       true,
			UploadedBy:     &userID,
		},
	}

	for _, img := range images {
		err := repo.Create(img)
		require.NoError(t, err)
		// Ensure distinct created_at values so DESC ordering assertions are
		// deterministic even on fast machines / low-resolution clocks.
		time.Sleep(2 * time.Millisecond)
	}

	t.Run("Count all images for industry news", func(t *testing.T) {
		count, err := repo.CountByIndustryNewsID(newsID)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("Count returns zero for industry news with no images", func(t *testing.T) {
		newNewsID := createTestIndustryNews(t, db)
		count, err := repo.CountByIndustryNewsID(newNewsID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestIndustryNewsImageRepository_CountActiveByIndustryNewsID(t *testing.T) {
	db := setupIndustryNewsImageTestDB(t)
	repo := NewIndustryNewsImageRepository(db)
	newsID := createTestIndustryNews(t, db)

	userID := uint(5)
	images := []*industry_news.IndustryNewsImage{
		{
			IndustryNewsID: newsID,
			ImageURL:       "https://example.com/image1.png",
			Title:          "Active Image 1",
			IsActive:       true,
			UploadedBy:     &userID,
		},
		{
			IndustryNewsID: newsID,
			ImageURL:       "https://example.com/image2.png",
			Title:          "Inactive Image",
			IsActive:       false,
			UploadedBy:     &userID,
		},
		{
			IndustryNewsID: newsID,
			ImageURL:       "https://example.com/image3.png",
			Title:          "Active Image 2",
			IsActive:       true,
			UploadedBy:     &userID,
		},
	}

	for _, img := range images {
		err := repo.Create(img)
		require.NoError(t, err)
		// Ensure distinct created_at values so DESC ordering assertions are
		// deterministic even on fast machines / low-resolution clocks.
		time.Sleep(2 * time.Millisecond)
	}

	t.Run("Count only active images", func(t *testing.T) {
		count, err := repo.CountActiveByIndustryNewsID(newsID)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("Count returns zero when no active images", func(t *testing.T) {
		newNewsID := createTestIndustryNews(t, db)
		inactiveImage := &industry_news.IndustryNewsImage{
			IndustryNewsID: newNewsID,
			ImageURL:       "https://example.com/inactive.png",
			Title:          "Inactive",
			IsActive:       false,
			UploadedBy:     &userID,
		}
		err := repo.Create(inactiveImage)
		require.NoError(t, err)

		count, err := repo.CountActiveByIndustryNewsID(newNewsID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}
