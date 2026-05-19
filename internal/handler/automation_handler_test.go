package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/bomanarakasura/jira-email-automation/internal/domain"
)

// MockAutomationService is a mock for testing automation handler
type MockAutomationService struct {
	mock.Mock
}

func (m *MockAutomationService) CreateRule(ctx interface{}, rule *domain.AutomationRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockAutomationService) GetRule(ctx interface{}, id string) (*domain.AutomationRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AutomationRule), args.Error(1)
}

func (m *MockAutomationService) GetUserRules(ctx interface{}, userID string) ([]*domain.AutomationRule, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AutomationRule), args.Error(1)
}

func (m *MockAutomationService) UpdateRule(ctx interface{}, rule *domain.AutomationRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockAutomationService) DeleteRule(ctx interface{}, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAutomationService) PauseRule(ctx interface{}, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAutomationService) ResumeRule(ctx interface{}, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAutomationService) GetActiveRules(ctx interface{}) ([]*domain.AutomationRule, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AutomationRule), args.Error(1)
}

func (m *MockAutomationService) TriggerRule(ctx interface{}, automationID string) error {
	args := m.Called(ctx, automationID)
	return args.Error(0)
}

// Helper function to create a test automation rule
func createTestAutomationRule() *domain.AutomationRule {
	return &domain.AutomationRule{
		ID:           "test-automation-id",
		UserID:       "test-user-123",
		JiraTicketID: "ticket-456",
		JiraTicketKey: "PROJ-123",
		Recipients:   []string{"test@example.com"},
		CronSchedule: "0 9 * * 1",
		Status:       domain.AutomationStatusActive,
	}
}

// Helper function to set up echo context with user ID and recorder
func setupEchoContextWithUser(e *echo.Echo, req *http.Request, userID string) (echo.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)

	// Extract URL parameters from the request path and set them in the context
	// This simulates what Echo's router does automatically
	pathParts := splitPath(req.URL.Path)
	if len(pathParts) > 0 {
		// For paths like /automations/test-automation-id, set the "id" parameter
		if len(pathParts) == 2 && pathParts[0] == "automations" {
			c.SetParamNames("id")
			c.SetParamValues(pathParts[1])
		}
		// For paths like /automations/test-automation-id/trigger
		if len(pathParts) == 3 && pathParts[0] == "automations" && pathParts[2] == "trigger" {
			c.SetParamNames("id")
			c.SetParamValues(pathParts[1])
		}
	}

	return c, rec
}

// Helper function to split URL path
func splitPath(path string) []string {
	// Remove leading slash and split
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}

func TestCreateAutomation_Success(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockAutomationService)
	handler := NewAutomationHandler(mockService)

	mockService.On("CreateRule", mock.Anything, mock.MatchedBy(func(r *domain.AutomationRule) bool {
		return r.UserID == "test-user-123" && r.JiraTicketKey == "PROJ-123"
	})).Return(nil)

	reqBody := CreateAutomationRequest{
		JiraTicketID:  "ticket-456",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com"},
		CronSchedule:  "0 9 * * 1",
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/automations", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	// Execute
	err := handler.CreateAutomation(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response domain.AutomationRule
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "PROJ-123", response.JiraTicketKey)
	assert.Equal(t, "test-user-123", response.UserID)

	mockService.AssertExpectations(t)
}

func TestCreateAutomation_InvalidCron(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockAutomationService)
	handler := NewAutomationHandler(mockService)

	mockService.On("CreateRule", mock.Anything, mock.Anything).Return(errors.New("invalid cron expression"))

	reqBody := CreateAutomationRequest{
		JiraTicketID:  "ticket-456",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com"},
		CronSchedule:  "invalid-cron",
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/automations", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	// Execute
	err := handler.CreateAutomation(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_CRON", response.Error.Code)

	mockService.AssertExpectations(t)
}

func TestCreateAutomation_EmptyRecipients(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockAutomationService)
	handler := NewAutomationHandler(mockService)

	reqBody := CreateAutomationRequest{
		JiraTicketID:  "ticket-456",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{}, // Empty recipients
		CronSchedule:  "0 9 * * 1",
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/automations", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	// Execute
	err := handler.CreateAutomation(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "MISSING_RECIPIENTS", response.Error.Code)

	mockService.AssertNotCalled(t, "CreateRule", mock.Anything, mock.Anything)
}

func TestCreateAutomation_MissingAuthToken(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockAutomationService)
	handler := NewAutomationHandler(mockService)

	reqBody := CreateAutomationRequest{
		JiraTicketID:  "ticket-456",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com"},
		CronSchedule:  "0 9 * * 1",
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/automations", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	// Context without user ID (not authenticated)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := handler.CreateAutomation(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "UNAUTHORIZED", response.Error.Code)

	mockService.AssertNotCalled(t, "CreateRule", mock.Anything, mock.Anything)
}

func TestListAutomations_Success(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockAutomationService)
	handler := NewAutomationHandler(mockService)

	expectedRules := []*domain.AutomationRule{
		createTestAutomationRule(),
		{
			ID:           "test-automation-id-2",
			UserID:       "test-user-123",
			JiraTicketID: "ticket-789",
			JiraTicketKey: "PROJ-456",
			Recipients:   []string{"another@example.com"},
			CronSchedule: "0 17 * * 5",
			Status:       domain.AutomationStatusActive,
		},
	}

	mockService.On("GetUserRules", mock.Anything, "test-user-123").Return(expectedRules, nil)

	req := httptest.NewRequest(http.MethodGet, "/automations", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	// Execute
	err := handler.ListAutomations(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "automations")

	mockService.AssertExpectations(t)
}

func TestGetAutomation_Success(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockAutomationService)
	handler := NewAutomationHandler(mockService)

	expectedRule := createTestAutomationRule()
	mockService.On("GetRule", mock.Anything, "test-automation-id").Return(expectedRule, nil)

	req := httptest.NewRequest(http.MethodGet, "/automations/test-automation-id", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	// Execute
	err := handler.GetAutomation(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response domain.AutomationRule
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test-automation-id", response.ID)
	assert.Equal(t, "PROJ-123", response.JiraTicketKey)

	mockService.AssertExpectations(t)
}

func TestGetAutomation_NotFound(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockAutomationService)
	handler := NewAutomationHandler(mockService)

	mockService.On("GetRule", mock.Anything, "non-existent-id").Return(nil, errors.New("automation rule not found"))

	req := httptest.NewRequest(http.MethodGet, "/automations/non-existent-id", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	// Execute
	err := handler.GetAutomation(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "AUTOMATION_NOT_FOUND", response.Error.Code)

	mockService.AssertExpectations(t)
}

func TestGetAutomation_Forbidden_OwnerMismatch(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockAutomationService)
	handler := NewAutomationHandler(mockService)

	// Rule belongs to different user
	differentUserRule := createTestAutomationRule()
	differentUserRule.UserID = "different-user-456"

	mockService.On("GetRule", mock.Anything, "test-automation-id").Return(differentUserRule, nil)

	req := httptest.NewRequest(http.MethodGet, "/automations/test-automation-id", nil)
	// Current user is test-user-123, but rule belongs to different-user-456
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	// Execute
	err := handler.GetAutomation(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "FORBIDDEN", response.Error.Code)

	mockService.AssertExpectations(t)
}

func TestUpdateAutomation_Success(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockAutomationService)
	handler := NewAutomationHandler(mockService)

	existingRule := createTestAutomationRule()
	mockService.On("GetRule", mock.Anything, "test-automation-id").Return(existingRule, nil)
	mockService.On("UpdateRule", mock.Anything, mock.MatchedBy(func(r *domain.AutomationRule) bool {
		return r.Recipients[0] == "updated@example.com"
	})).Return(nil)

	reqBody := UpdateAutomationRequest{
		Recipients: []string{"updated@example.com"},
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, "/automations/test-automation-id", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	// Execute
	err := handler.UpdateAutomation(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response domain.AutomationRule
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "updated@example.com", response.Recipients[0])

	mockService.AssertExpectations(t)
}

func TestDeleteAutomation_Success(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockAutomationService)
	handler := NewAutomationHandler(mockService)

	existingRule := createTestAutomationRule()
	mockService.On("GetRule", mock.Anything, "test-automation-id").Return(existingRule, nil)
	mockService.On("DeleteRule", mock.Anything, "test-automation-id").Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/automations/test-automation-id", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	// Execute
	err := handler.DeleteAutomation(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	mockService.AssertExpectations(t)
}

func TestDeleteAutomation_NotFound(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockAutomationService)
	handler := NewAutomationHandler(mockService)

	mockService.On("GetRule", mock.Anything, "non-existent-id").Return(nil, errors.New("automation rule not found"))

	req := httptest.NewRequest(http.MethodDelete, "/automations/non-existent-id", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	// Execute
	err := handler.DeleteAutomation(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "AUTOMATION_NOT_FOUND", response.Error.Code)

	mockService.AssertExpectations(t)
}

func TestTriggerAutomation_Success(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockAutomationService)
	handler := NewAutomationHandler(mockService)

	existingRule := createTestAutomationRule()
	mockService.On("GetRule", mock.Anything, "test-automation-id").Return(existingRule, nil)
	mockService.On("TriggerRule", mock.Anything, "test-automation-id").Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/automations/test-automation-id/trigger", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	// Execute
	err := handler.TriggerAutomation(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "automation rule triggered successfully", response["message"])
	assert.Equal(t, "test-automation-id", response["automation_id"])

	mockService.AssertExpectations(t)
}

func TestTriggerAutomation_NotFound(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockAutomationService)
	handler := NewAutomationHandler(mockService)

	mockService.On("GetRule", mock.Anything, "non-existent-id").Return(nil, errors.New("automation rule not found"))

	req := httptest.NewRequest(http.MethodPost, "/automations/non-existent-id/trigger", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	// Execute
	err := handler.TriggerAutomation(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "AUTOMATION_NOT_FOUND", response.Error.Code)

	mockService.AssertExpectations(t)
}