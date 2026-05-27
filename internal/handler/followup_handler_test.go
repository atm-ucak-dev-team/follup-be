package handler

import (
	"bytes"
	"context"
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

type MockJiraServiceV2 struct {
	mock.Mock
}

func (m *MockJiraServiceV2) GetTicket(ctx interface{}, ticketID string) (*domain.JiraTicket, error) {
	args := m.Called(ctx, ticketID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.JiraTicket), args.Error(1)
}

func (m *MockJiraServiceV2) UpdateTicketStatus(ctx interface{}, ticketID, status string) error {
	args := m.Called(ctx, ticketID, status)
	return args.Error(0)
}

func (m *MockJiraServiceV2) AddComment(ctx interface{}, ticketID, comment string) error {
	args := m.Called(ctx, ticketID, comment)
	return args.Error(0)
}

func (m *MockJiraServiceV2) GetAuthenticatedUser(ctx interface{}) (*domain.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockJiraServiceV2) GetIssues(ctx context.Context, cloudID, accessToken, search, limit string) ([]domain.JiraIssueResponse, error) {
	args := m.Called(ctx, cloudID, accessToken, search, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.JiraIssueResponse), args.Error(1)
}

func (m *MockJiraServiceV2) GetIssue(ctx context.Context, cloudID, accessToken, issueID string) (*domain.JiraIssueDetailResponse, error) {
	args := m.Called(ctx, cloudID, accessToken, issueID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.JiraIssueDetailResponse), args.Error(1)
}

type MockEmailThreadRepository struct {
	mock.Mock
}

func (m *MockEmailThreadRepository) Create(ctx context.Context, thread *domain.EmailThread) error {
	args := m.Called(ctx, thread)
	return args.Error(0)
}

func (m *MockEmailThreadRepository) GetByID(ctx context.Context, id string) (*domain.EmailThread, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.EmailThread), args.Error(1)
}

func (m *MockEmailThreadRepository) GetByAutomationID(ctx context.Context, automationID string) ([]*domain.EmailThread, error) {
	args := m.Called(ctx, automationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.EmailThread), args.Error(1)
}

func (m *MockEmailThreadRepository) GetByGmailThreadID(ctx context.Context, gmailThreadID string) (*domain.EmailThread, error) {
	args := m.Called(ctx, gmailThreadID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.EmailThread), args.Error(1)
}

func (m *MockEmailThreadRepository) Update(ctx context.Context, thread *domain.EmailThread) error {
	args := m.Called(ctx, thread)
	return args.Error(0)
}

func (m *MockEmailThreadRepository) UpdateThreadStatus(ctx context.Context, threadID, status string) error {
	args := m.Called(ctx, threadID, status)
	return args.Error(0)
}

func (m *MockEmailThreadRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

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

func (m *MockFollowupServiceV2) GetFollowupDetail(ctx interface{}, id string) (*service2.FollowupDetail, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service2.FollowupDetail), args.Error(1)
}

func (m *MockFollowupServiceV2) GetSummary(ctx interface{}, userID string, jiraTicketID string) (*service2.FollowupSummary, error) {
	args := m.Called(ctx, userID, jiraTicketID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service2.FollowupSummary), args.Error(1)
}

func (m *MockFollowupServiceV2) GetGlobalSummary(ctx interface{}, userID string) (*service2.FollowupSummary, error) {
	args := m.Called(ctx, userID)
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
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

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
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

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
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

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
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

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

func TestCreateFollowup_Success(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

	mockSvc.On("CreateRule", mock.Anything, mock.MatchedBy(func(r *domain.Followup) bool {
		return r.UserID == "test-user-123" && r.JiraTicketID == "10001" && r.Repeat == 3 && r.FollowupConfirmation
	})).Return(nil)

	reqBody := CreateFollowupRequest{
		JiraTicketID:         "10001",
		JiraTicketTitle:      "Test Ticket Title",
		JiraStakeholder:      "John Doe",
		JiraTicketStatus:     "In Progress",
		To:                   "test@example.com",
		Cc:                   "cc@example.com",
		Subject:              "Test Subject",
		EmailBody:            "Test body content",
		StartDateTime:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpireDateTime:       time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		Frequency:            "0 9 * * 1",
		Repeat:               3,
		FollowupConfirmation: true,
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/followups", bytes.NewReader(bodyJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.CreateFollowup(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response domain.Followup
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "10001", response.JiraTicketID)
	assert.Equal(t, "test@example.com", response.To)
	assert.Equal(t, "Test Subject", response.Subject)
	assert.Equal(t, "0 9 * * 1", response.Frequency)
	assert.Equal(t, 3, response.Repeat)
	assert.True(t, response.FollowupConfirmation)

	mockSvc.AssertExpectations(t)
}

func TestCreateFollowup_InvalidFrequency(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

	mockSvc.On("CreateRule", mock.Anything, mock.Anything).Return(errors.New("invalid frequency: cron parse error"))

	reqBody := CreateFollowupRequest{
		JiraTicketID:     "10001",
		JiraTicketTitle:  "Test Ticket Title",
		JiraStakeholder:  "John Doe",
		JiraTicketStatus: "In Progress",
		To:               "test@example.com",
		Subject:          "Test Subject",
		EmailBody:        "Test body",
		Frequency:        "bad-cron",
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/followups", bytes.NewReader(bodyJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.CreateFollowup(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_FREQUENCY", response.Error.Code)

	mockSvc.AssertExpectations(t)
}

func TestCreateFollowup_MissingFields(t *testing.T) {
	tests := []struct {
		name    string
		body    CreateFollowupRequest
		errCode string
	}{
		{
			name: "missing jiraTicketId",
			body: CreateFollowupRequest{
				To: "test@example.com", Subject: "S", EmailBody: "B", Frequency: "0 9 * * 1",
			},
			errCode: "MISSING_JIRA_TICKET_ID",
		},
		{
			name: "missing to",
			body: CreateFollowupRequest{
				JiraTicketID: "10001", Subject: "S", EmailBody: "B", Frequency: "0 9 * * 1",
			},
			errCode: "MISSING_TO",
		},
		{
			name: "missing subject",
			body: CreateFollowupRequest{
				JiraTicketID: "10001", To: "test@example.com", EmailBody: "B", Frequency: "0 9 * * 1",
			},
			errCode: "MISSING_SUBJECT",
		},
		{
			name: "missing emailBody",
			body: CreateFollowupRequest{
				JiraTicketID: "10001", To: "test@example.com", Subject: "S", Frequency: "0 9 * * 1",
			},
			errCode: "MISSING_EMAIL_BODY",
		},
		{
			name: "missing frequency",
			body: CreateFollowupRequest{
				JiraTicketID: "10001", To: "test@example.com", Subject: "S", EmailBody: "B",
			},
			errCode: "MISSING_FREQUENCY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			mockSvc := new(MockFollowupServiceV2)
			mockEmailThreadRepo := new(MockEmailThreadRepository)
			handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

			bodyJSON, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/followups", bytes.NewReader(bodyJSON))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			c, rec := setupEchoContextWithUser(e, req, "test-user-123")

			err := handler.CreateFollowup(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var response ErrorResponse
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.errCode, response.Error.Code)

			mockSvc.AssertNotCalled(t, "CreateRule", mock.Anything, mock.Anything)
		})
	}
}

func TestCreateFollowup_MissingAuthToken(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

	reqBody := CreateFollowupRequest{
		JiraTicketID:     "10001",
		JiraTicketTitle:  "Test Ticket Title",
		JiraStakeholder:  "John Doe",
		JiraTicketStatus: "In Progress",
		To:               "test@example.com",
		Subject:          "Test Subject",
		EmailBody:        "Test body",
		Frequency:        "0 9 * * 1",
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/followups", bytes.NewReader(bodyJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.CreateFollowup(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "UNAUTHORIZED", response.Error.Code)

	mockSvc.AssertNotCalled(t, "CreateRule", mock.Anything, mock.Anything)
}

func TestCreateFollowup_ServiceError(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

	mockSvc.On("CreateRule", mock.Anything, mock.Anything).Return(errors.New("db error"))

	reqBody := CreateFollowupRequest{
		JiraTicketID:     "10001",
		JiraTicketTitle:  "Test Ticket Title",
		JiraStakeholder:  "John Doe",
		JiraTicketStatus: "In Progress",
		To:               "test@example.com",
		Subject:          "Test Subject",
		EmailBody:        "Test body",
		Frequency:        "0 9 * * 1",
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/followups", bytes.NewReader(bodyJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.CreateFollowup(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "FOLLOWUP_CREATE_FAILED", response.Error.Code)

	mockSvc.AssertExpectations(t)
}

func TestCreateFollowup_InvalidToEmail(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

	reqBody := CreateFollowupRequest{
		JiraTicketID:     "10001",
		JiraTicketTitle:  "Test Ticket Title",
		JiraStakeholder:  "John Doe",
		JiraTicketStatus: "In Progress",
		To:               "not-an-email",
		Subject:          "Test Subject",
		EmailBody:        "Test body",
		Frequency:        "0 9 * * 1",
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/followups", bytes.NewReader(bodyJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.CreateFollowup(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_TO", response.Error.Code)

	mockSvc.AssertNotCalled(t, "CreateRule", mock.Anything, mock.Anything)
}

func TestCreateFollowup_InvalidCcEmail(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

	reqBody := CreateFollowupRequest{
		JiraTicketID:     "10001",
		JiraTicketTitle:  "Test Ticket Title",
		JiraStakeholder:  "John Doe",
		JiraTicketStatus: "In Progress",
		To:               "test@example.com",
		Cc:               "not-an-email",
		Subject:          "Test Subject",
		EmailBody:        "Test body",
		Frequency:        "0 9 * * 1",
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/followups", bytes.NewReader(bodyJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.CreateFollowup(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_CC", response.Error.Code)

	mockSvc.AssertNotCalled(t, "CreateRule", mock.Anything, mock.Anything)
}

func TestCreateFollowup_SubjectTooLong(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

	reqBody := CreateFollowupRequest{
		JiraTicketID:     "10001",
		JiraTicketTitle:  "Test Ticket Title",
		JiraStakeholder:  "John Doe",
		JiraTicketStatus: "In Progress",
		To:               "test@example.com",
		Subject:          string(make([]byte, 501)),
		EmailBody:        "Test body",
		Frequency:        "0 9 * * 1",
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/followups", bytes.NewReader(bodyJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.CreateFollowup(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_SUBJECT", response.Error.Code)

	mockSvc.AssertNotCalled(t, "CreateRule", mock.Anything, mock.Anything)
}

func TestCreateFollowup_EmailBodyTooLong(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

	reqBody := CreateFollowupRequest{
		JiraTicketID:     "10001",
		JiraTicketTitle:  "Test Ticket Title",
		JiraStakeholder:  "John Doe",
		JiraTicketStatus: "In Progress",
		To:               "test@example.com",
		Subject:          "Test Subject",
		EmailBody:        string(make([]byte, 10001)),
		Frequency:        "0 9 * * 1",
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/followups", bytes.NewReader(bodyJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.CreateFollowup(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_EMAIL_BODY", response.Error.Code)

	mockSvc.AssertNotCalled(t, "CreateRule", mock.Anything, mock.Anything)
}

func TestCreateFollowup_NegativeRepeat(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

	reqBody := CreateFollowupRequest{
		JiraTicketID:     "10001",
		JiraTicketTitle:  "Test Ticket Title",
		JiraStakeholder:  "John Doe",
		JiraTicketStatus: "In Progress",
		To:               "test@example.com",
		Subject:          "Test Subject",
		EmailBody:        "Test body",
		Frequency:        "0 9 * * 1",
		Repeat:           -1,
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/followups", bytes.NewReader(bodyJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.CreateFollowup(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REPEAT", response.Error.Code)

	mockSvc.AssertNotCalled(t, "CreateRule", mock.Anything, mock.Anything)
}

func TestGetSummary_Success(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

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
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

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
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

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
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

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

func TestGetGlobalSummary_Success(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

	expectedSummary := &service2.FollowupSummary{
		JiraTicketID: "",
		JiraTitle:    "",
		Replied:      3,
		Ongoing:      4,
		Expired:      2,
	}

	mockSvc.On("GetGlobalSummary", mock.Anything, "test-user-123").Return(expectedSummary, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/statistic", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.GetGlobalSummary(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response StatisticResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 3, response.Replied)
	assert.Equal(t, 4, response.Ongoing)
	assert.Equal(t, 2, response.Expired)

	mockSvc.AssertExpectations(t)
}

func TestGetGlobalSummary_Unauthorized(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/statistic", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetGlobalSummary(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "UNAUTHORIZED", response.Error.Code)

	mockSvc.AssertNotCalled(t, "GetGlobalSummary", mock.Anything, mock.Anything)
}

func TestGetGlobalSummary_ServiceError(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

	mockSvc.On("GetGlobalSummary", mock.Anything, "test-user-123").Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/statistic", nil)
	c, rec := setupEchoContextWithUser(e, req, "test-user-123")

	err := handler.GetGlobalSummary(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "GLOBAL_SUMMARY_FAILED", response.Error.Code)

	mockSvc.AssertExpectations(t)
}

func TestGetFollowupsByTicketID_Success(t *testing.T) {
	e := echo.New()
	mockSvc := new(MockFollowupServiceV2)
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

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
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

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
	mockEmailThreadRepo := new(MockEmailThreadRepository)
	handler := NewFollowupHandler(mockSvc, new(MockJiraServiceV2), mockEmailThreadRepo)

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
