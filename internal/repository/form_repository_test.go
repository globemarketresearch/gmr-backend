package repository

import (
	"testing"
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/form"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create table
	err = db.AutoMigrate(&form.FormSubmission{})
	require.NoError(t, err)

	return db
}

func TestFormRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFormRepository(db)

	submission1 := &form.FormSubmission{
		Category: form.CategoryContact,
		Status:   form.StatusPending,
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
		Category: form.CategoryRequestSample,
		Status:   form.StatusPending,
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

	t.Run("Successfully retrieve submission by ID", func(t *testing.T) {
		result, err := repo.GetByID(submission1.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, submission1.ID, result.ID)
		assert.Equal(t, form.CategoryContact, result.Category)
	})

	t.Run("Retrieve second submission by ID", func(t *testing.T) {
		result, err := repo.GetByID(submission2.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, submission2.ID, result.ID)
		assert.Equal(t, form.CategoryRequestSample, result.Category)
	})

	t.Run("Return error for non-existent ID", func(t *testing.T) {
		result, err := repo.GetByID(999999)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestFormRepository_GetAll_WithCategoryFilter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFormRepository(db)

	submissions := []*form.FormSubmission{
		{
			Category: form.CategoryContact,
			Status:   form.StatusPending,
			Data: form.FormData{
				"fullName": "Test User 1",
				"email":    "test1@example.com",
				"company":  "Company 1",
				"subject":  "Subject 1",
				"message":  "Message 1",
			},
		},
		{
			Category: form.CategoryRequestSample,
			Status:   form.StatusProcessed,
			Data: form.FormData{
				"fullName":    "Test User 2",
				"email":       "test2@example.com",
				"company":     "Company 2",
				"jobTitle":    "Manager",
				"reportTitle": "Report 2",
			},
		},
	}

	for _, s := range submissions {
		err := repo.Create(s)
		require.NoError(t, err)
	}

	t.Run("Filter by category", func(t *testing.T) {
		query := form.GetSubmissionsQuery{
			Category: string(form.CategoryContact),
			Page:     1,
			Limit:    10,
		}

		results, total, err := repo.GetAll(query)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, results, 1)
		assert.Equal(t, form.CategoryContact, results[0].Category)
	})

	t.Run("No category filter returns all", func(t *testing.T) {
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

func TestFormRepository_GetAll_SortByCreatedAt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFormRepository(db)

	// CreatedAt is set explicitly (and spaced apart) so ordering assertions
	// below are deterministic regardless of clock resolution; GORM only
	// auto-populates CreatedAt when it is left as the zero value.
	base := time.Now().Truncate(time.Second)
	submissions := []*form.FormSubmission{
		{
			Category:  form.CategoryContact,
			Status:    form.StatusPending,
			CreatedAt: base,
			Data: form.FormData{
				"fullName": "User A",
				"email":    "a@example.com",
				"company":  "Company A",
				"subject":  "Subject A",
				"message":  "Message A",
			},
		},
		{
			Category:  form.CategoryContact,
			Status:    form.StatusPending,
			CreatedAt: base.Add(1 * time.Second),
			Data: form.FormData{
				"fullName": "User B",
				"email":    "b@example.com",
				"company":  "Company B",
				"subject":  "Subject B",
				"message":  "Message B",
			},
		},
		{
			Category:  form.CategoryContact,
			Status:    form.StatusPending,
			CreatedAt: base.Add(2 * time.Second),
			Data: form.FormData{
				"fullName": "User C",
				"email":    "c@example.com",
				"company":  "Company C",
				"subject":  "Subject C",
				"message":  "Message C",
			},
		},
	}

	for _, s := range submissions {
		err := repo.Create(s)
		require.NoError(t, err)
	}

	t.Run("Sort by createdAt ascending", func(t *testing.T) {
		query := form.GetSubmissionsQuery{
			Page:      1,
			Limit:     10,
			SortBy:    "createdAt",
			SortOrder: "asc",
		}

		results, total, err := repo.GetAll(query)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, results, 3)
		assert.Equal(t, submissions[0].ID, results[0].ID)
		assert.Equal(t, submissions[1].ID, results[1].ID)
		assert.Equal(t, submissions[2].ID, results[2].ID)
	})

	t.Run("Sort by createdAt descending", func(t *testing.T) {
		query := form.GetSubmissionsQuery{
			Page:      1,
			Limit:     10,
			SortBy:    "createdAt",
			SortOrder: "desc",
		}

		results, total, err := repo.GetAll(query)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, results, 3)
		assert.Equal(t, submissions[2].ID, results[0].ID)
		assert.Equal(t, submissions[1].ID, results[1].ID)
		assert.Equal(t, submissions[0].ID, results[2].ID)
	})
}
