package cron

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/atm-ucak/follup/internal/frequency"
	"github.com/atm-ucak/follup/internal/repository"
	"github.com/atm-ucak/follup/internal/service"
	"github.com/robfig/cron/v3"
)

// Scheduler manages followup rule execution using cron
type Scheduler struct {
	cron            *cron.Cron
	followupRepo    repository.FollowupRepository
	emailService    service.EmailService
	emailThreadRepo repository.EmailThreadRepository
	entryIDs        map[string]cron.EntryID
	mu              sync.RWMutex
}

// NewScheduler creates a new cron scheduler instance
func NewScheduler(followupRepo repository.FollowupRepository, emailService service.EmailService, emailThreadRepo repository.EmailThreadRepository) *Scheduler {
	return &Scheduler{
		cron:            cron.New(),
		followupRepo:    followupRepo,
		emailService:    emailService,
		emailThreadRepo: emailThreadRepo,
		entryIDs:        make(map[string]cron.EntryID),
	}
}

// Start initializes the scheduler with active followup rules and begins execution
func (s *Scheduler) Start() error {
	log.Println("Starting cron scheduler...")

	ctx := context.Background()
	activeRules, err := s.followupRepo.GetActiveRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to load active followup rules: %w", err)
	}

	log.Printf("Loaded %d active followup rules", len(activeRules))

	for _, rule := range activeRules {
		if err := s.AddRule(*rule); err != nil {
			log.Printf("Error adding rule %s: %v", rule.ID, err)
		} else {
			log.Printf("Added rule %s with schedule %s", rule.ID, rule.Frequency)
		}
	}

	// Register periodic sync job to pick up database changes
	_, err = s.cron.AddFunc("@every 1m", func() {
		if err := s.SyncRules(); err != nil {
			log.Printf("Error syncing rules: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register sync job: %w", err)
	}
	log.Println("Registered periodic sync job (every 1m)")

	// Register reply detection job (every 2 minutes)
	_, err = s.cron.AddFunc("@every 1m", func() {
		if err := s.pollReplies(); err != nil {
			log.Printf("Error checking for replies: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register reply detection job: %w", err)
	}
	log.Println("Registered reply detection job (every 2m)")

	s.cron.Start()
	log.Println("Cron scheduler started successfully")

	return nil
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	log.Println("Stopping cron scheduler...")

	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := s.cron.Stop()

	<-ctx.Done()

	log.Println("Cron scheduler stopped")
}

// AddRule dynamically adds a new followup rule to the scheduler
func (s *Scheduler) AddRule(rule domain.Followup) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.entryIDs[rule.ID]; exists {
		return fmt.Errorf("rule %s already exists in scheduler", rule.ID)
	}

	// Convert frequency to cron expression if needed
	cronExpr := rule.Frequency
	if frequency.IsConvertibleFrequency(rule.Frequency) {
		var err error
		cronExpr, err = frequency.FrequencyToCron(rule.Frequency, rule.StartDateTime)
		if err != nil {
			return fmt.Errorf("failed to convert frequency for scheduling: %w", err)
		}
		log.Printf("Converted frequency '%s' to cron expression '%s' for rule %s",
			rule.Frequency, cronExpr, rule.ID)
	}

	jobFunc := func() {
		s.executeAutomation(rule.ID)
	}

	entryID, err := s.cron.AddFunc(cronExpr, jobFunc)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.entryIDs[rule.ID] = entryID
	log.Printf("Added rule %s with schedule %s", rule.ID, cronExpr)

	return nil
}

// RemoveRule dynamically removes a followup rule from the scheduler
func (s *Scheduler) RemoveRule(automationID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, exists := s.entryIDs[automationID]
	if !exists {
		log.Printf("Rule %s not found in scheduler", automationID)
		return
	}

	s.cron.Remove(entryID)

	delete(s.entryIDs, automationID)

	log.Printf("Removed rule %s from scheduler", automationID)
}

// executeAutomation executes the followup rule and handles errors
func (s *Scheduler) executeAutomation(automationID string) {
	log.Printf("Executing followup %s", automationID)

	ctx := context.Background()

	// Get the latest rule data to check expiration conditions
	rule, err := s.followupRepo.GetByID(ctx, automationID)
	if err != nil {
		log.Printf("Error getting followup %s: %v", automationID, err)
		return
	}

	// Check if repeat limit has been reached
	if rule.ExecutionCount >= rule.Repeat {
		log.Printf("Skipping followup %s: repeat limit reached (%d/%d)",
			automationID, rule.ExecutionCount, rule.Repeat)
		s.RemoveRule(automationID)
		return
	}

	// Check if expired
	if time.Now().After(rule.ExpireDateTime) {
		log.Printf("Skipping followup %s: expired at %v",
			automationID, rule.ExpireDateTime)
		s.RemoveRule(automationID)
		return
	}

	// Safety check - skip if startDateTime hasn't been reached
	if rule.StartDateTime.After(time.Now()) {
		log.Printf("Skipping followup %s: startDateTime not reached (starts at %v)",
			automationID, rule.StartDateTime)
		return
	}

	if err := s.emailService.SendFollowUpByAutomation(ctx, automationID); err != nil {
		log.Printf("Error executing followup %s: %v", automationID, err)
		return
	}

	// Re-fetch rule to check if it was marked as expired during execution
	updatedRule, err := s.followupRepo.GetByID(ctx, automationID)
	if err != nil {
		log.Printf("Error getting updated followup %s: %v", automationID, err)
		return
	}

	// If rule was marked as expired during execution, remove from scheduler
	if updatedRule.Status == domain.FollowupStatusExpired {
		log.Printf("Followup %s expired and removed from scheduler", automationID)
		s.RemoveRule(automationID)
		return
	}

	log.Printf("Successfully executed followup %s", automationID)
}

// SyncRules compares database state with scheduler state and updates accordingly
// This ensures new, deleted, or inactive followups are picked up dynamically
func (s *Scheduler) SyncRules() error {
	log.Println("Syncing followup rules with scheduler...")

	ctx := context.Background()
	activeRules, err := s.followupRepo.GetActiveRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to load active followup rules for sync: %w", err)
	}

	// Build map of DB rule IDs for O(1) lookup
	dbRuleIDs := make(map[string]bool, len(activeRules))
	for _, rule := range activeRules {
		dbRuleIDs[rule.ID] = true
	}

	// Use read lock to check current state
	s.mu.RLock()
	currentRuleIDs := make(map[string]bool, len(s.entryIDs))
	for ruleID := range s.entryIDs {
		currentRuleIDs[ruleID] = true
	}
	s.mu.RUnlock()

	addedCount := 0
	removedCount := 0

	// Find and add new rules (in DB but not in scheduler)
	for _, rule := range activeRules {
		if _, exists := currentRuleIDs[rule.ID]; !exists {
			if err := s.AddRule(*rule); err != nil {
				log.Printf("Error adding rule %s during sync: %v", rule.ID, err)
			} else {
				addedCount++
			}
		}
	}

	// Find and remove deleted/inactive rules (in scheduler but not in DB)
	for ruleId := range currentRuleIDs {
		if _, exists := dbRuleIDs[ruleId]; !exists {
			s.RemoveRule(ruleId)
			removedCount++
		}
	}

	log.Printf("Sync completed: %d active rules, %d added, %d removed",
		len(activeRules), addedCount, removedCount)

	return nil
}

// GetScheduledRuleCount returns the number of currently scheduled rules
func (s *Scheduler) GetScheduledRuleCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entryIDs)
}

// pollReplies checks email inboxes for replies to ongoing followups
func (s *Scheduler) pollReplies() error {
	ctx := context.Background()
	return s.emailService.PollInbox(ctx)
}
