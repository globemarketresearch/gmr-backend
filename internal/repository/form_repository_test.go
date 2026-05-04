package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/healthcare-market-research/backend/internal/domain/form"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Helper function to create uint64 pointers
func uint64Ptr(v uint64) *uint64 {
	return &v
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create table
	err = db.AutoMigrate(&form.FormSubmission{})
	require.NoError(t, err)

	// Note: SQLite doesn't support sequences like PostgreSQL
	// For testing purposes, we'll set tracking numbers manually
	return db
}

func TestFormRepository_GetByTrackingNumber(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFormRepository(db)

	// Create test submissions
	submission1 := &form.FormSubmission{
		ID:             "sub_" + uuid.New().String(),
		TrackingNumber: uint64Ptr(1),
		Category:       form.CategoryContact,
		Status:         form.StatusPending,
		Data: form.FormData{
			"fullName": "John Doe",
			"email":    "john@example.com",
			"company":  "Test Corp",
			"subject":  "Test Subject",
			"message":  "Test Message",
		},
	}

	err := repo.Create(submission1)
	require.NoError(t, err)

	submission2 := &form.FormSubmission{
		ID:             "sub_" + uuid.New().String(),
		TrackingNumber: uint64Ptr(2),
		Category:       form.CategoryRequestSample,
		Status:         form.StatusPending,
		Data: form.FormData{
			"fullName":    "Jane Smith",
			"email":       "jane@example.com",
			"company":     "Sample Inc",
			"jobTitle":    "Manager",
			"reportTitle": "Healthcare Report 2024",
		},
	}

	err = repo.Create(submission2)
	require.NoError(t, err)

	// Test GetByTrackingNumber
	t.Run("Successfully retrieve submission by tracking number", func(t *testing.T) {
		result, err := repo.GetByTrackingNumber(1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, uint64(1), *result.TrackingNumber)
		assert.Equal(t, submission1.ID, result.ID)
		assert.Equal(t, form.CategoryContact, result.Category)
	})

	t.Run("Retrieve second submission by tracking number", func(t *testing.T) {
		result, err := repo.GetByTrackingNumber(2)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, uint64(2), *result.TrackingNumber)
		assert.Equal(t, submission2.ID, result.ID)
		assert.Equal(t, form.CategoryRequestSample, result.Category)
	})

	t.Run("Return error for non-existent tracking number", func(t *testing.T) {
		result, err := repo.GetByTrackingNumber(999)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestFormRepository_GetAll_WithTrackingNumberFilter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFormRepository(db)

	// Create test submissions
	submissions := []*form.FormSubmission{
		{
			ID:             "sub_" + uuid.New().String(),
			TrackingNumber: uint64Ptr(1),
			Category:       form.CategoryContact,
			Status:         form.StatusPending,
			Data: form.FormData{
				"fullName": "Test User 1",
				"email":    "test1@example.com",
				"company":  "Company 1",
				"subject":  "Subject 1",
				"message":  "Message 1",
			},
		},
		{
			ID:             "sub_" + uuid.New().String(),
			TrackingNumber: uint64Ptr(2),
			Category:       form.CategoryContact,
			Status:         form.StatusProcessed,
			Data: form.FormData{
				"fullName": "Test User 2",
				"email":    "test2@example.com",
				"company":  "Company 2",
				"subject":  "Subject 2",
				"message":  "Message 2",
			},
		},
	}

	for _, s := range submissions {
		err := repo.Create(s)
		require.NoError(t, err)
	}

	t.Run("Filter by tracking number", func(t *testing.T) {
		query := form.GetSubmissionsQuery{
			TrackingNumber: uint64Ptr(1),
			Page:           1,
			Limit:          10,
		}

		results, total, err := repo.GetAll(query)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, results, 1)
		assert.Equal(t, uint64(1), *results[0].TrackingNumber)
	})

	t.Run("No tracking number filter returns all", func(t *testing.T) {
		query := form.GetSubmissionsQuery{
			Page:  1,
			Limit: 10,
		}

		results, total, err := repo.GetAll(query)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, results, 2)
	})
}

func TestFormRepository_GetAll_SortByTrackingNumber(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFormRepository(db)

	// Create test submissions with different tracking numbers
	submissions := []*form.FormSubmission{
		{
			ID:             "sub_" + uuid.New().String(),
			TrackingNumber: uint64Ptr(3),
			Category:       form.CategoryContact,
			Status:         form.StatusPending,
			Data: form.FormData{
				"fullName": "User C",
				"email":    "c@example.com",
				"company":  "Company C",
				"subject":  "Subject C",
				"message":  "Message C",
			},
		},
		{
			ID:             "sub_" + uuid.New().String(),
			TrackingNumber: uint64Ptr(1),
			Category:       form.CategoryContact,
			Status:         form.StatusPending,
			Data: form.FormData{
				"fullName": "User A",
				"email":    "a@example.com",
				"company":  "Company A",
				"subject":  "Subject A",
				"message":  "Message A",
			},
		},
		{
			ID:             "sub_" + uuid.New().String(),
			TrackingNumber: uint64Ptr(2),
			Category:       form.CategoryContact,
			Status:         form.StatusPending,
			Data: form.FormData{
				"fullName": "User B",
				"email":    "b@example.com",
				"company":  "Company B",
				"subject":  "Subject B",
				"message":  "Message B",
			},
		},
	}

	for _, s := range submissions {
		err := repo.Create(s)
		require.NoError(t, err)
	}

	t.Run("Sort by tracking number ascending", func(t *testing.T) {
		query := form.GetSubmissionsQuery{
			Page:      1,
			Limit:     10,
			SortBy:    "trackingNumber",
			SortOrder: "asc",
		}

		results, total, err := repo.GetAll(query)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, results, 3)
		assert.Equal(t, uint64(1), *results[0].TrackingNumber)
		assert.Equal(t, uint64(2), *results[1].TrackingNumber)
		assert.Equal(t, uint64(3), *results[2].TrackingNumber)
	})

	t.Run("Sort by tracking number descending", func(t *testing.T) {
		query := form.GetSubmissionsQuery{
			Page:      1,
			Limit:     10,
			SortBy:    "trackingNumber",
			SortOrder: "desc",
		}

		results, total, err := repo.GetAll(query)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, results, 3)
		assert.Equal(t, uint64(3), *results[0].TrackingNumber)
		assert.Equal(t, uint64(2), *results[1].TrackingNumber)
		assert.Equal(t, uint64(1), *results[2].TrackingNumber)
	})
}
