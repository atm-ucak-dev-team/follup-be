package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	service2 "github.com/atm-ucak/follup/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockFollowupServiceV2 struct {
	mock.Mock
}

func (m *MockFollowupServiceV2) CreateRule(ctx interface{}, rule *domain.Followup) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockFollowupServiceV2) GetRule(ctx interface{}, id string) (*domain.Followup, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Followup), args.Error(1)
}

func (m *MockFollowupServiceV2) GetUserRules(ctx interface{}, userID string) ([]*domain.Followup, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Followup), args.Error(1)
}

func (m *MockFollowupServiceV2) UpdateRule(ctx interface{}, rule *domain.Followup) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockFollowupServiceV2) DeleteRule(ctx interface{}, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFollowupServiceV2) PauseRule(ctx interface{}, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFollowupServiceV2) ResumeRule(ctx interface{}, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFollowupServiceV2) GetActiveRules(ctx interface{}) ([]*domain.Followup, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Followup), args.Error(1)
}

func (m *MockFollowupServiceV2) TriggerRule(ctx interface{}, automationID string) error {
	args := m.Called(ctx, automationID)
	return args.Error(0)
}

func (m *MockFollowupServiceV2) ListFollowups(ctx interface{}, userID string, jiraTicketID string) ([]*domain.Followup, error) {
	args := m.Called(ctx, userID, jiraTicketID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Followup), args.Error(1)
}

func (m *MockFollowupServiceV2) ListFollowupDetails(ctx interface{}, userID string, jiraTicketID string) ([]*service2.FollowupDetail, error) {
	args := m.Called(ctx, userID, jiraTicketID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*service2.FollowupDetail), args.Error(1)
}

func (m *MockFollowupServiceV2) GetSummary(ctx interface{}, userID string, jiraTicketID string) (*service2.FollowupSummary, error) {
	args := m.Called(ctx, userID, jiraTicketID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service2.FollowupSummary), args.Error(1)
}

func createTestFollowupDetail(status string) *service2.FollowupDetail {
	now := time.Now()
	f := &domain.Followup{
		ID:            "test-followup-1",
		JiraTicketID:  "10001",
		JiraTicketKey: "PROJ-123",
		UserID:        "test-user-123",
		To:            "test@example.com",
		Subject:       "Test Subject",
		EmailBody:     "Test body content",
		Status:        domain.FollowupStatusOngoing,
		Frequency:     "0 9 * * 1",
		LastRunAt:     &now,
		CreatedAt:     now,
	}
	d := &service2.FollowupDetail{
		Followup:        f,
		EffectiveStatus: status,
	}
	switch status {
	case "ongoing":
		next := now.Add(24 * time.Hour)
		d.NextFollowUp = &next
	case "replied":
		d.RepliedAt = &now
	case "expired":
		// lastFollowUp derived from Followup.LastRunAt
	}
	return d
}

func TestListFollowups_Success(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	handler := NewFollowupHandler(mockSvc)

	expected := []*service2.FollowupDetail{
		createTestFollowupDetail("ongoing"),
	}

	mockSvc.On("ListFollowupDetails", mock.Anything, "test-user-123", "").Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/followup", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.ListFollowups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response []FollowupItem
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 1)
	assert.Equal(t, "test-followup-1", response[0].FollowupID)
	assert.Equal(t, "PROJ-123", response[0].JiraTicketID)
	assert.Equal(t, "Test Subject", response[0].Subject)
	assert.Equal(t, "ongoing", response[0].Status)
	assert.NotNil(t, response[0].NextFollowUp)
	assert.Nil(t, response[0].RepliedAt)
	assert.Nil(t, response[0].LastFollowUp)

	mockSvc.AssertExpectations(t)
}

func TestListFollowups_WithJiraTicketFilter(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	handler := NewFollowupHandler(mockSvc)

	expected := []*service2.FollowupDetail{
		createTestFollowupDetail("ongoing"),
	}

	mockSvc.On("ListFollowupDetails", mock.Anything, "test-user-123", "PROJ-123").Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/followup?jiraTicket=PROJ-123", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.ListFollowups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response []FollowupItem
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 1)
	assert.Equal(t, "PROJ-123", response[0].JiraTicketID)
	assert.Equal(t, "ongoing", response[0].Status)

	mockSvc.AssertExpectations(t)
}

func TestListFollowups_Unauthorized(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	handler := NewFollowupHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/followup", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ListFollowups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "UNAUTHORIZED", response.Error.Code)

	mockSvc.AssertNotCalled(t, "ListFollowupDetails", mock.Anything, mock.Anything, mock.Anything)
}

func TestListFollowups_ServiceError(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	handler := NewFollowupHandler(mockSvc)

	mockSvc.On("ListFollowupDetails", mock.Anything, "test-user-123", "").Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/followup", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.ListFollowups(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "FOLLOWUP_LIST_FAILED", response.Error.Code)

	mockSvc.AssertExpectations(t)
}

func TestGetSummary_Success(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	handler := NewFollowupHandler(mockSvc)

	expectedSummary := &service2.FollowupSummary{
		JiraTicketID: "PROJ-123",
		JiraTitle:    "",
		Replied:      2,
		Ongoing:      1,
		Expired:      0,
	}

	mockSvc.On("GetSummary", mock.Anything, "test-user-123", "PROJ-123").Return(expectedSummary, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/PROJ-123/summary", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")
	c.SetParamNames("jiraTicketID")
	c.SetParamValues("PROJ-123")

	err := handler.GetSummary(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response FollowupSummaryResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "PROJ-123", response.JiraTicketID)
	assert.Equal(t, 2, response.Replied)
	assert.Equal(t, 1, response.Ongoing)
	assert.Equal(t, 0, response.Expired)

	mockSvc.AssertExpectations(t)
}

func TestGetSummary_Unauthorized(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	handler := NewFollowupHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/PROJ-123/summary", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetSummary(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "UNAUTHORIZED", response.Error.Code)

	mockSvc.AssertNotCalled(t, "GetSummary", mock.Anything, mock.Anything, mock.Anything)
}

func TestGetSummary_MissingTicketID(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	handler := NewFollowupHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1//summary", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.GetSummary(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "MISSING_JIRA_TICKET_ID", response.Error.Code)

	mockSvc.AssertNotCalled(t, "GetSummary", mock.Anything, mock.Anything, mock.Anything)
}

func TestGetSummary_ServiceError(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	handler := NewFollowupHandler(mockSvc)

	mockSvc.On("GetSummary", mock.Anything, "test-user-123", "PROJ-123").Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/PROJ-123/summary", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")
	c.SetParamNames("jiraTicketID")
	c.SetParamValues("PROJ-123")

	err := handler.GetSummary(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "SUMMARY_FAILED", response.Error.Code)

	mockSvc.AssertExpectations(t)
}

func TestGetFollowupsByTicketID_Success(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	handler := NewFollowupHandler(mockSvc)

	expected := []*service2.FollowupDetail{
		createTestFollowupDetail("ongoing"),
	}

	mockSvc.On("ListFollowupDetails", mock.Anything, "test-user-123", "PROJ-123").Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/PROJ-123/followups", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")
	c.SetParamNames("jiraTicketID")
	c.SetParamValues("PROJ-123")

	err := handler.GetFollowupsByTicketID(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response []FollowupItem
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 1)
	assert.Equal(t, "test-followup-1", response[0].FollowupID)
	assert.Equal(t, "ongoing", response[0].Status)

	mockSvc.AssertExpectations(t)
}

func TestGetFollowupsByTicketID_MissingTicketID(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	handler := NewFollowupHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1//followups", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.GetFollowupsByTicketID(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "MISSING_JIRA_TICKET_ID", response.Error.Code)

	mockSvc.AssertNotCalled(t, "ListFollowupDetails", mock.Anything, mock.Anything, mock.Anything)
}

func TestGetFollowupsByTicketID_Unauthorized(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	handler := NewFollowupHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/PROJ-123/followups", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetFollowupsByTicketID(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "UNAUTHORIZED", response.Error.Code)

	mockSvc.AssertNotCalled(t, "ListFollowupDetails", mock.Anything, mock.Anything, mock.Anything)
}
