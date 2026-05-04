package repository

import (
	"testing"

	"github.com/healthcare-market-research/backend/internal/domain/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupReportImageTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create tables
	err = db.AutoMigrate(&report.Report{}, &report.ReportImage{})
	require.NoError(t, err)

	return db
}

func createTestReport(t *testing.T, db *gorm.DB) uint {
	testReport := &report.Report{
		Title:       "Test Report",
		Description: "Test Description",
		Status:      report.StatusDraft,
	}
	err := db.Create(testReport).Error
	require.NoError(t, err)
	return testReport.ID
}

func TestReportImageRepository_Create(t *testing.T) {
	db := setupReportImageTestDB(t)
	repo := NewReportImageRepository(db)
	reportID := createTestReport(t, db)

	userID := uint(5)
	image := &report.ReportImage{
		ReportID:   reportID,
		ImageURL:   "https://example.com/image1.png",
		Title:      "Test Image",
		IsActive:   true,
		UploadedBy: &userID,
	}

	err := repo.Create(image)
	require.NoError(t, err)
	assert.NotZero(t, image.ID)
	assert.NotZero(t, image.CreatedAt)
	assert.NotZero(t, image.UpdatedAt)
}

func TestReportImageRepository_FindByID(t *testing.T) {
	db := setupReportImageTestDB(t)
	repo := NewReportImageRepository(db)
	reportID := createTestReport(t, db)

	userID := uint(5)
	image := &report.ReportImage{
		ReportID:   reportID,
		ImageURL:   "https://example.com/image1.png",
		Title:      "Test Image",
		IsActive:   true,
		UploadedBy: &userID,
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

func TestReportImageRepository_FindByReportID(t *testing.T) {
	db := setupReportImageTestDB(t)
	repo := NewReportImageRepository(db)
	reportID := createTestReport(t, db)

	userID := uint(5)
	images := []*report.ReportImage{
		{
			ReportID:   reportID,
			ImageURL:   "https://example.com/image1.png",
			Title:      "Image 1",
			IsActive:   true,
			UploadedBy: &userID,
		},
		{
			ReportID:   reportID,
			ImageURL:   "https://example.com/image2.png",
			Title:      "Image 2",
			IsActive:   false,
			UploadedBy: &userID,
		},
		{
			ReportID:   reportID,
			ImageURL:   "https://example.com/image3.png",
			Title:      "Image 3",
			IsActive:   true,
			UploadedBy: &userID,
		},
	}

	for _, img := range images {
		err := repo.Create(img)
		require.NoError(t, err)
	}

	t.Run("Retrieve all images for report", func(t *testing.T) {
		results, err := repo.FindByReportID(reportID)
		require.NoError(t, err)
		assert.Len(t, results, 3)
		// Should be ordered by created_at DESC (newest first)
		assert.Equal(t, "Image 3", results[0].Title)
		assert.Equal(t, "Image 2", results[1].Title)
		assert.Equal(t, "Image 1", results[2].Title)
	})

	t.Run("Return empty array for report with no images", func(t *testing.T) {
		newReportID := createTestReport(t, db)
		results, err := repo.FindByReportID(newReportID)
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})
}

func TestReportImageRepository_FindActiveByReportID(t *testing.T) {
	db := setupReportImageTestDB(t)
	repo := NewReportImageRepository(db)
	reportID := createTestReport(t, db)

	userID := uint(5)
	images := []*report.ReportImage{
		{
			ReportID:   reportID,
			ImageURL:   "https://example.com/image1.png",
			Title:      "Active Image 1",
			IsActive:   true,
			UploadedBy: &userID,
		},
		{
			ReportID:   reportID,
			ImageURL:   "https://example.com/image2.png",
			Title:      "Inactive Image",
			IsActive:   false,
			UploadedBy: &userID,
		},
		{
			ReportID:   reportID,
			ImageURL:   "https://example.com/image3.png",
			Title:      "Active Image 2",
			IsActive:   true,
			UploadedBy: &userID,
		},
	}

	for _, img := range images {
		err := repo.Create(img)
		require.NoError(t, err)
	}

	t.Run("Retrieve only active images", func(t *testing.T) {
		results, err := repo.FindActiveByReportID(reportID)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		assert.True(t, results[0].IsActive)
		assert.True(t, results[1].IsActive)
		// Should be ordered by created_at DESC
		assert.Equal(t, "Active Image 2", results[0].Title)
		assert.Equal(t, "Active Image 1", results[1].Title)
	})

	t.Run("Return empty array when no active images", func(t *testing.T) {
		newReportID := createTestReport(t, db)
		inactiveImage := &report.ReportImage{
			ReportID:   newReportID,
			ImageURL:   "https://example.com/inactive.png",
			Title:      "Inactive",
			IsActive:   false,
			UploadedBy: &userID,
		}
		err := repo.Create(inactiveImage)
		require.NoError(t, err)

		results, err := repo.FindActiveByReportID(newReportID)
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})
}

func TestReportImageRepository_Update(t *testing.T) {
	db := setupReportImageTestDB(t)
	repo := NewReportImageRepository(db)
	reportID := createTestReport(t, db)

	userID := uint(5)
	image := &report.ReportImage{
		ReportID:   reportID,
		ImageURL:   "https://example.com/image1.png",
		Title:      "Original Title",
		IsActive:   true,
		UploadedBy: &userID,
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

func TestReportImageRepository_Delete(t *testing.T) {
	db := setupReportImageTestDB(t)
	repo := NewReportImageRepository(db)
	reportID := createTestReport(t, db)

	userID := uint(5)
	image := &report.ReportImage{
		ReportID:   reportID,
		ImageURL:   "https://example.com/image1.png",
		Title:      "Test Image",
		IsActive:   true,
		UploadedBy: &userID,
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
	activeImages, err := repo.FindActiveByReportID(reportID)
	require.NoError(t, err)
	assert.Len(t, activeImages, 0)

	// Verify it doesn't appear in all results either
	allImages, err := repo.FindByReportID(reportID)
	require.NoError(t, err)
	assert.Len(t, allImages, 0)
}

func TestReportImageRepository_CountByReportID(t *testing.T) {
	db := setupReportImageTestDB(t)
	repo := NewReportImageRepository(db)
	reportID := createTestReport(t, db)

	userID := uint(5)
	images := []*report.ReportImage{
		{
			ReportID:   reportID,
			ImageURL:   "https://example.com/image1.png",
			Title:      "Image 1",
			IsActive:   true,
			UploadedBy: &userID,
		},
		{
			ReportID:   reportID,
			ImageURL:   "https://example.com/image2.png",
			Title:      "Image 2",
			IsActive:   false,
			UploadedBy: &userID,
		},
		{
			ReportID:   reportID,
			ImageURL:   "https://example.com/image3.png",
			Title:      "Image 3",
			IsActive:   true,
			UploadedBy: &userID,
		},
	}

	for _, img := range images {
		err := repo.Create(img)
		require.NoError(t, err)
	}

	t.Run("Count all images for report", func(t *testing.T) {
		count, err := repo.CountByReportID(reportID)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("Count returns zero for report with no images", func(t *testing.T) {
		newReportID := createTestReport(t, db)
		count, err := repo.CountByReportID(newReportID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestReportImageRepository_CountActiveByReportID(t *testing.T) {
	db := setupReportImageTestDB(t)
	repo := NewReportImageRepository(db)
	reportID := createTestReport(t, db)

	userID := uint(5)
	images := []*report.ReportImage{
		{
			ReportID:   reportID,
			ImageURL:   "https://example.com/image1.png",
			Title:      "Active Image 1",
			IsActive:   true,
			UploadedBy: &userID,
		},
		{
			ReportID:   reportID,
			ImageURL:   "https://example.com/image2.png",
			Title:      "Inactive Image",
			IsActive:   false,
			UploadedBy: &userID,
		},
		{
			ReportID:   reportID,
			ImageURL:   "https://example.com/image3.png",
			Title:      "Active Image 2",
			IsActive:   true,
			UploadedBy: &userID,
		},
	}

	for _, img := range images {
		err := repo.Create(img)
		require.NoError(t, err)
	}

	t.Run("Count only active images", func(t *testing.T) {
		count, err := repo.CountActiveByReportID(reportID)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("Count returns zero when no active images", func(t *testing.T) {
		newReportID := createTestReport(t, db)
		inactiveImage := &report.ReportImage{
			ReportID:   newReportID,
			ImageURL:   "https://example.com/inactive.png",
			Title:      "Inactive",
			IsActive:   false,
			UploadedBy: &userID,
		}
		err := repo.Create(inactiveImage)
		require.NoError(t, err)

		count, err := repo.CountActiveByReportID(newReportID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}
