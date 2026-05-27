package service

import (
	"context"
	"fmt"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/atm-ucak/follup/internal/repository"
	"github.com/robfig/cron/v3"
)

// AutomationServiceImpl implements the AutomationService interface
type AutomationServiceImpl struct {
	followupRepo    repository.FollowupRepository
	emailThreadRepo repository.EmailThreadRepository
	emailService    EmailService
}

// NewAutomationService creates a new AutomationService instance
func NewAutomationService(
	followupRepo repository.FollowupRepository,
	emailThreadRepo repository.EmailThreadRepository,
	emailService EmailService,
) AutomationService {
	return &AutomationServiceImpl{
		followupRepo:    followupRepo,
		emailThreadRepo: emailThreadRepo,
		emailService:    emailService,
	}
}

// CreateRule creates a new followup rule with validation
func (s *AutomationServiceImpl) CreateRule(ctx interface{}, rule *domain.Followup) error {
	contextCast, ok := ctx.(context.Context)
	if !ok {
		return fmt.Errorf("invalid context type")
	}

	if err := s.validateRule(rule); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if rule.Status == "" {
		rule.Status = domain.FollowupStatusOngoing
	}

	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}

	if err := s.followupRepo.Create(contextCast, rule); err != nil {
		return fmt.Errorf("failed to create followup rule: %w", err)
	}

	return nil
}

// GetRule retrieves a followup rule by ID
func (s *AutomationServiceImpl) GetRule(ctx interface{}, id string) (*domain.Followup, error) {
	contextCast, ok := ctx.(context.Context)
	if !ok {
		return nil, fmt.Errorf("invalid context type")
	}

	if id == "" {
		return nil, fmt.Errorf("followup ID cannot be empty")
	}

	rule, err := s.followupRepo.GetByID(contextCast, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get followup rule: %w", err)
	}

	return rule, nil
}

// GetUserRules retrieves all followup rules for a specific user
func (s *AutomationServiceImpl) GetUserRules(ctx interface{}, userID string) ([]*domain.Followup, error) {
	contextCast, ok := ctx.(context.Context)
	if !ok {
		return nil, fmt.Errorf("invalid context type")
	}

	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	rules, err := s.followupRepo.GetByUserID(contextCast, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user followup rules: %w", err)
	}

	return rules, nil
}

// UpdateRule updates an existing followup rule with ownership validation
func (s *AutomationServiceImpl) UpdateRule(ctx interface{}, rule *domain.Followup) error {
	contextCast, ok := ctx.(context.Context)
	if !ok {
		return fmt.Errorf("invalid context type")
	}

	if rule.ID == "" {
		return fmt.Errorf("followup ID cannot be empty")
	}

	existingRule, err := s.followupRepo.GetByID(contextCast, rule.ID)
	if err != nil {
		return fmt.Errorf("failed to get existing followup rule: %w", err)
	}

	if existingRule.UserID != rule.UserID {
		return fmt.Errorf("user does not own this followup rule")
	}

	if err := s.validateRule(rule); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := s.followupRepo.Update(contextCast, rule); err != nil {
		return fmt.Errorf("failed to update followup rule: %w", err)
	}

	return nil
}

// DeleteRule deletes a followup rule with ownership validation
func (s *AutomationServiceImpl) DeleteRule(ctx interface{}, id string) error {
	contextCast, ok := ctx.(context.Context)
	if !ok {
		return fmt.Errorf("invalid context type")
	}

	if id == "" {
		return fmt.Errorf("followup ID cannot be empty")
	}

	rule, err := s.followupRepo.GetByID(contextCast, id)
	if err != nil {
		return fmt.Errorf("failed to get followup rule: %w", err)
	}

	if rule == nil {
		return fmt.Errorf("followup rule not found")
	}

	if err := s.followupRepo.Delete(contextCast, id); err != nil {
		return fmt.Errorf("failed to delete followup rule: %w", err)
	}

	return nil
}

// PauseRule pauses an active followup rule
func (s *AutomationServiceImpl) PauseRule(ctx interface{}, id string) error {
	contextCast, ok := ctx.(context.Context)
	if !ok {
		return fmt.Errorf("invalid context type")
	}

	if id == "" {
		return fmt.Errorf("followup ID cannot be empty")
	}

	rule, err := s.followupRepo.GetByID(contextCast, id)
	if err != nil {
		return fmt.Errorf("failed to get followup rule: %w", err)
	}

	rule.Status = domain.FollowupStatusStopped

	if err := s.followupRepo.Update(contextCast, rule); err != nil {
		return fmt.Errorf("failed to pause followup rule: %w", err)
	}

	return nil
}

// ResumeRule resumes a paused followup rule
func (s *AutomationServiceImpl) ResumeRule(ctx interface{}, id string) error {
	contextCast, ok := ctx.(context.Context)
	if !ok {
		return fmt.Errorf("invalid context type")
	}

	if id == "" {
		return fmt.Errorf("followup ID cannot be empty")
	}

	rule, err := s.followupRepo.GetByID(contextCast, id)
	if err != nil {
		return fmt.Errorf("failed to get followup rule: %w", err)
	}

	rule.Status = domain.FollowupStatusOngoing

	if err := s.followupRepo.Update(contextCast, rule); err != nil {
		return fmt.Errorf("failed to resume followup rule: %w", err)
	}

	return nil
}

// GetFollowupDetail retrieves a single followup with computed status and timestamps
func (s *AutomationServiceImpl) GetFollowupDetail(ctx interface{}, id string) (*FollowupDetail, error) {
	contextCast, ok := ctx.(context.Context)
	if !ok {
		return nil, fmt.Errorf("invalid context type")
	}

	if id == "" {
		return nil, fmt.Errorf("followup ID cannot be empty")
	}

	rule, err := s.followupRepo.GetByID(contextCast, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get followup rule: %w", err)
	}

	d := &FollowupDetail{Followup: rule}
	s.enrichFollowupDetail(contextCast, d)
	return d, nil
}

// GetActiveRules retrieves all active followup rules
func (s *AutomationServiceImpl) GetActiveRules(ctx interface{}) ([]*domain.Followup, error) {
	contextCast, ok := ctx.(context.Context)
	if !ok {
		return nil, fmt.Errorf("invalid context type")
	}

	rules, err := s.followupRepo.GetActiveRules(contextCast)
	if err != nil {
		return nil, fmt.Errorf("failed to get active followup rules: %w", err)
	}

	return rules, nil
}

// TriggerRule manually executes a followup rule with ownership validation
func (s *AutomationServiceImpl) TriggerRule(ctx interface{}, automationID string) error {
	contextCast, ok := ctx.(context.Context)
	if !ok {
		return fmt.Errorf("invalid context type")
	}

	if automationID == "" {
		return fmt.Errorf("followup ID cannot be empty")
	}

	rule, err := s.followupRepo.GetByID(contextCast, automationID)
	if err != nil {
		return fmt.Errorf("failed to get followup rule: %w", err)
	}

	if rule == nil {
		return fmt.Errorf("followup rule not found")
	}

	if err := s.emailService.SendFollowUpByAutomation(contextCast, automationID); err != nil {
		return fmt.Errorf("failed to trigger followup rule: %w", err)
	}

	return nil
}

// ListFollowups retrieves followups for a user, optionally filtered by jira ticket
func (s *AutomationServiceImpl) ListFollowups(ctx interface{}, userID string, jiraTicketID string) ([]*domain.Followup, error) {
	contextCast, ok := ctx.(context.Context)
	if !ok {
		return nil, fmt.Errorf("invalid context type")
	}

	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	rules, err := s.followupRepo.GetByUserID(contextCast, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get followup rules: %w", err)
	}

	if jiraTicketID == "" {
		return rules, nil
	}

	filtered := make([]*domain.Followup, 0, len(rules))
	for _, r := range rules {
		if r.JiraTicketKey == jiraTicketID || r.JiraTicketID == jiraTicketID {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

// ListFollowupDetails retrieves followups with computed status and timestamps
func (s *AutomationServiceImpl) ListFollowupDetails(ctx interface{}, userID string, jiraTicketID string) ([]*FollowupDetail, error) {
	contextCast, ok := ctx.(context.Context)
	if !ok {
		return nil, fmt.Errorf("invalid context type")
	}

	rules, err := s.ListFollowups(ctx, userID, jiraTicketID)
	if err != nil {
		return nil, err
	}

	details := make([]*FollowupDetail, 0, len(rules))
	for _, r := range rules {
		d := &FollowupDetail{Followup: r}
		s.enrichFollowupDetail(contextCast, d)
		details = append(details, d)
	}

	return details, nil
}

// GetSummary returns summary counts for a specific jira ticket
func (s *AutomationServiceImpl) GetSummary(ctx interface{}, userID string, jiraTicketID string) (*FollowupSummary, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	if jiraTicketID == "" {
		return nil, fmt.Errorf("jira ticket ID cannot be empty")
	}

	details, err := s.ListFollowupDetails(ctx, userID, jiraTicketID)
	if err != nil {
		return nil, err
	}

	summary := &FollowupSummary{
		JiraTicketID: jiraTicketID,
		JiraTitle:    "",
	}

	for _, d := range details {
		switch d.EffectiveStatus {
		case "replied":
			summary.Replied++
		case "ongoing":
			summary.Ongoing++
		case "expired":
			summary.Expired++
		}
	}

	return summary, nil
}

// GetGlobalSummary returns summary counts across all jira tickets for a user
func (s *AutomationServiceImpl) GetGlobalSummary(ctx interface{}, userID string) (*FollowupSummary, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	details, err := s.ListFollowupDetails(ctx, userID, "")
	if err != nil {
		return nil, err
	}

	summary := &FollowupSummary{
		JiraTicketID: "",
		JiraTitle:    "",
	}

	for _, d := range details {
		switch d.EffectiveStatus {
		case "replied":
			summary.Replied++
		case "ongoing":
			summary.Ongoing++
		case "expired":
			summary.Expired++
		}
	}

	return summary, nil
}

// enrichFollowupDetail computes effective status and timestamps for a followup
func (s *AutomationServiceImpl) enrichFollowupDetail(ctx context.Context, d *FollowupDetail) {
	r := d.Followup
	now := time.Now()

	// Check for replied threads
	threads, err := s.emailThreadRepo.GetByAutomationID(ctx, r.ID)
	if err == nil {
		for _, t := range threads {
			if t.Status == domain.EmailThreadStatusReplied {
				d.EffectiveStatus = "replied"
				if d.RepliedAt == nil || t.LastSyncedAt.After(*d.RepliedAt) {
					d.RepliedAt = &t.LastSyncedAt
				}
			}
		}
	}

	if d.EffectiveStatus == "replied" {
		return
	}

	// Check if expired
	if r.ExpireDateTime.Before(now) {
		d.EffectiveStatus = "expired"
		return
	}

	// Default to ongoing + compute next follow-up
	d.EffectiveStatus = "ongoing"
	if r.Frequency != "" {
		schedule, err := cron.ParseStandard(r.Frequency)
		if err == nil {
			next := schedule.Next(now)
			d.NextFollowUp = &next
		}
	}
}

func (s *AutomationServiceImpl) validateRule(rule *domain.Followup) error {
	if rule == nil {
		return fmt.Errorf("rule cannot be nil")
	}

	if rule.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	if rule.JiraTicketID == "" {
		return fmt.Errorf("Jira ticket ID is required")
	}

	if rule.To == "" {
		return fmt.Errorf("recipient email (to) is required")
	}

	if rule.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if rule.EmailBody == "" {
		return fmt.Errorf("email body is required")
	}

	if rule.Frequency == "" {
		return fmt.Errorf("frequency is required")
	}

	if _, err := cron.ParseStandard(rule.Frequency); err != nil {
		return fmt.Errorf("invalid frequency: %w", err)
	}

	if rule.Status != "" &&
		rule.Status != domain.FollowupStatusOngoing &&
		rule.Status != domain.FollowupStatusCompleted &&
		rule.Status != domain.FollowupStatusStopped {
		return fmt.Errorf("status must be 'ongoing', 'completed', or 'stopped', got: %s", rule.Status)
	}

	return nil
}
