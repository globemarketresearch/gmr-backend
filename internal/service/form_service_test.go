package service

import (
	"errors"
	"testing"

	"github.com/healthcare-market-research/backend/internal/domain/form"
	"github.com/healthcare-market-research/backend/internal/domain/order"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockFormRepository is a mock implementation of repository.FormRepository.
type MockFormRepository struct {
	mock.Mock
}

func (m *MockFormRepository) Create(submission *form.FormSubmission) error {
	args := m.Called(submission)
	return args.Error(0)
}

func (m *MockFormRepository) GetAll(query form.GetSubmissionsQuery) ([]form.FormSubmission, int64, error) {
	args := m.Called(query)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]form.FormSubmission), int64(args.Int(1)), args.Error(2)
}

func (m *MockFormRepository) GetByID(id uint) (*form.FormSubmission, error) {
	args := m.Called(id)
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

func (m *MockFormRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockFormRepository) BulkDelete(ids []uint) (int64, error) {
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

func (m *MockFormRepository) UpdateStatus(id uint, status form.FormStatus, processedBy *uint) error {
	args := m.Called(id, status, processedBy)
	return args.Error(0)
}

// mockEmailService is a no-op implementation of email.EmailService for tests.
type mockEmailService struct{}

func (m *mockEmailService) SendFormNotification(submission *form.FormSubmission) error {
	return nil
}

func (m *mockEmailService) SendOrderConfirmation(o *order.Order) error {
	return nil
}

func (m *mockEmailService) SendOrderAdminNotification(o *order.Order) error {
	return nil
}

func TestFormService_Create_Success(t *testing.T) {
	mockRepo := new(MockFormRepository)
	service := NewFormService(mockRepo, &mockEmailService{})

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
	assert.Equal(t, form.CategoryContact, response.Category)

	mockRepo.AssertExpectations(t)
}

func TestFormService_Create_InvalidCategory(t *testing.T) {
	mockRepo := new(MockFormRepository)
	service := NewFormService(mockRepo, &mockEmailService{})

	req := &form.CreateSubmissionRequest{
		Category: form.FormCategory("not-a-real-category"),
		Data:     form.FormData{},
	}

	response, err := service.Create(req)

	assert.Error(t, err)
	assert.Nil(t, response)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything)
}

func TestFormService_GetByID(t *testing.T) {
	mockRepo := new(MockFormRepository)
	service := NewFormService(mockRepo, &mockEmailService{})

	expectedSubmission := &form.FormSubmission{
		ID:       42,
		Category: form.CategoryContact,
		Status:   form.StatusPending,
		Data: form.FormData{
			"fullName": "Jane Smith",
			"email":    "jane@example.com",
			"company":  "Sample Inc",
			"subject":  "Test",
			"message":  "Message",
		},
	}

	t.Run("Successfully retrieve submission by ID", func(t *testing.T) {
		mockRepo.On("GetByID", uint(42)).Return(expectedSubmission, nil).Once()

		result, err := service.GetByID(42)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedSubmission.ID, result.ID)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Return error when submission not found", func(t *testing.T) {
		mockRepo.On("GetByID", uint(999)).Return(nil, errors.New("record not found")).Once()

		result, err := service.GetByID(999)

		assert.Error(t, err)
		assert.Nil(t, result)

		mockRepo.AssertExpectations(t)
	})
}

func TestFormService_Delete(t *testing.T) {
	mockRepo := new(MockFormRepository)
	service := NewFormService(mockRepo, &mockEmailService{})

	t.Run("Delete succeeds", func(t *testing.T) {
		mockRepo.On("Delete", uint(1)).Return(nil).Once()

		err := service.Delete(1)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Delete handles error gracefully", func(t *testing.T) {
		mockRepo.On("Delete", uint(2)).Return(errors.New("database error")).Once()

		err := service.Delete(2)

		assert.Error(t, err)
		assert.Equal(t, "database error", err.Error())
		mockRepo.AssertExpectations(t)
	})
}

func TestFormService_UpdateStatus(t *testing.T) {
	mockRepo := new(MockFormRepository)
	service := NewFormService(mockRepo, &mockEmailService{})

	t.Run("UpdateStatus succeeds for a valid status", func(t *testing.T) {
		mockRepo.On("UpdateStatus", uint(1), form.StatusProcessed, (*uint)(nil)).Return(nil).Once()

		err := service.UpdateStatus(1, form.StatusProcessed, nil)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("UpdateStatus validates status", func(t *testing.T) {
		err := service.UpdateStatus(1, form.FormStatus("invalid"), nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
		mockRepo.AssertNotCalled(t, "UpdateStatus", uint(1), form.FormStatus("invalid"), (*uint)(nil))
	})
}
