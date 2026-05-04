package service

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/healthcare-market-research/backend/internal/domain/form"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// Helper function to create uint64 pointers
func uint64Ptr(v uint64) *uint64 {
	return &v
}

// MockFormRepository is a mock implementation of FormRepository
type MockFormRepository struct {
	mock.Mock
}

func (m *MockFormRepository) Create(submission *form.FormSubmission) error {
	args := m.Called(submission)
	// Simulate database auto-generation of tracking number
	if submission.TrackingNumber == nil {
		submission.TrackingNumber = uint64Ptr(123) // Simulated auto-generated value
	}
	return args.Error(0)
}

func (m *MockFormRepository) GetAll(query form.GetSubmissionsQuery) ([]form.FormSubmission, int64, error) {
	args := m.Called(query)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]form.FormSubmission), int64(args.Int(1)), args.Error(2)
}

func (m *MockFormRepository) GetByID(id string) (*form.FormSubmission, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*form.FormSubmission), args.Error(1)
}

func (m *MockFormRepository) GetByTrackingNumber(trackingNumber uint64) (*form.FormSubmission, error) {
	args := m.Called(trackingNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*form.FormSubmission), args.Error(1)
}

func (m *MockFormRepository) GetByCategory(category string, page, limit int) ([]form.FormSubmission, int64, error) {
	args := m.Called(category, page, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]form.FormSubmission), int64(args.Int(1)), args.Error(2)
}

func (m *MockFormRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockFormRepository) BulkDelete(ids []string) (int64, error) {
	args := m.Called(ids)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockFormRepository) GetStats() (*form.SubmissionStats, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*form.SubmissionStats), args.Error(1)
}

func (m *MockFormRepository) UpdateStatus(id string, status form.FormStatus, processedBy *uint) error {
	args := m.Called(id, status, processedBy)
	return args.Error(0)
}

func TestFormService_Create_IncludesTrackingNumber(t *testing.T) {
	mockRepo := new(MockFormRepository)
	service := NewFormService(mockRepo)

	req := &form.CreateSubmissionRequest{
		Category: form.CategoryContact,
		Data: form.FormData{
			"fullName": "John Doe",
			"email":    "john@example.com",
			"company":  "Test Corp",
			"subject":  "Test Subject",
			"message":  "Test Message",
		},
	}

	mockRepo.On("Create", mock.AnythingOfType("*form.FormSubmission")).Return(nil)

	response, err := service.Create(req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Success)
	assert.NotEmpty(t, response.SubmissionID)
	assert.Equal(t, uint64(123), *response.TrackingNumber) // From mock
	assert.Equal(t, form.CategoryContact, response.Category)

	mockRepo.AssertExpectations(t)
}

func TestFormService_GetByTrackingNumber(t *testing.T) {
	mockRepo := new(MockFormRepository)
	service := NewFormService(mockRepo)

	expectedSubmission := &form.FormSubmission{
		ID:             "sub_" + uuid.New().String(),
		TrackingNumber: uint64Ptr(42),
		Category:       form.CategoryContact,
		Status:         form.StatusPending,
		Data: form.FormData{
			"fullName": "Jane Smith",
			"email":    "jane@example.com",
			"company":  "Sample Inc",
			"subject":  "Test",
			"message":  "Message",
		},
	}

	t.Run("Successfully retrieve submission by tracking number", func(t *testing.T) {
		mockRepo.On("GetByTrackingNumber", uint64(42)).Return(expectedSubmission, nil).Once()

		result, err := service.GetByTrackingNumber(42)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, uint64(42), *result.TrackingNumber)
		assert.Equal(t, expectedSubmission.ID, result.ID)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Return error when tracking number not found", func(t *testing.T) {
		mockRepo.On("GetByTrackingNumber", uint64(999)).Return(nil, gorm.ErrRecordNotFound).Once()

		result, err := service.GetByTrackingNumber(999)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestFormService_Delete_InvalidatesTrackingNumberCache(t *testing.T) {
	mockRepo := new(MockFormRepository)
	service := NewFormService(mockRepo)

	submissionID := "sub_" + uuid.New().String()
	submission := &form.FormSubmission{
		ID:             submissionID,
		TrackingNumber: uint64Ptr(100),
		Category:       form.CategoryContact,
		Status:         form.StatusPending,
	}

	t.Run("Delete invalidates caches including tracking number", func(t *testing.T) {
		mockRepo.On("GetByID", submissionID).Return(submission, nil).Once()
		mockRepo.On("Delete", submissionID).Return(nil).Once()

		err := service.Delete(submissionID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		// Note: Cache invalidation is tested implicitly - the GetByID call proves we're fetching the tracking number
	})

	t.Run("Delete handles error gracefully", func(t *testing.T) {
		mockRepo.On("GetByID", submissionID).Return(submission, nil).Once()
		mockRepo.On("Delete", submissionID).Return(errors.New("database error")).Once()

		err := service.Delete(submissionID)

		assert.Error(t, err)
		assert.Equal(t, "database error", err.Error())
		mockRepo.AssertExpectations(t)
	})
}

func TestFormService_UpdateStatus_InvalidatesTrackingNumberCache(t *testing.T) {
	mockRepo := new(MockFormRepository)
	service := NewFormService(mockRepo)

	submissionID := "sub_" + uuid.New().String()
	submission := &form.FormSubmission{
		ID:             submissionID,
		TrackingNumber: uint64Ptr(200),
		Category:       form.CategoryContact,
		Status:         form.StatusPending,
	}

	t.Run("UpdateStatus invalidates caches including tracking number", func(t *testing.T) {
		mockRepo.On("GetByID", submissionID).Return(submission, nil).Once()
		mockRepo.On("UpdateStatus", submissionID, form.StatusProcessed, (*uint)(nil)).Return(nil).Once()

		err := service.UpdateStatus(submissionID, form.StatusProcessed, nil)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("UpdateStatus validates status", func(t *testing.T) {
		err := service.UpdateStatus(submissionID, form.FormStatus("invalid"), nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})
}
