package cron

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/bomanarakasura/jira-email-automation/internal/domain"
	"github.com/bomanarakasura/jira-email-automation/internal/repository"
	"github.com/bomanarakasura/jira-email-automation/internal/service"
)

// Scheduler manages automation rule execution using cron
type Scheduler struct {
	cron           *cron.Cron
	automationRepo repository.AutomationRuleRepository
	emailService   service.EmailService
	entryIDs       map[string]cron.EntryID // automationID -> cron entry ID
	mu             sync.RWMutex
}

// NewScheduler creates a new cron scheduler instance
func NewScheduler(automationRepo repository.AutomationRuleRepository, emailService service.EmailService) *Scheduler {
	return &Scheduler{
		cron:           cron.New(), // Default supports 5-field cron expressions (minute-level granularity)
		automationRepo: automationRepo,
		emailService:   emailService,
		entryIDs:       make(map[string]cron.EntryID),
	}
}

// Start initializes the scheduler with active automation rules and begins execution
func (s *Scheduler) Start() error {
	log.Println("Starting cron scheduler...")

	// Load all active automation rules from repository
	ctx := context.Background()
	activeRules, err := s.automationRepo.GetActiveRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to load active automation rules: %w", err)
	}

	log.Printf("Loaded %d active automation rules", len(activeRules))

	// Register each active rule with the cron scheduler
	for _, rule := range activeRules {
		if err := s.AddRule(*rule); err != nil {
			log.Printf("Error adding rule %s: %v", rule.ID, err)
			// Continue loading other rules even if one fails
		} else {
			log.Printf("Added rule %s with schedule %s", rule.ID, rule.CronSchedule)
		}
	}

	// Start the cron runner
	s.cron.Start()
	log.Println("Cron scheduler started successfully")

	return nil
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	log.Println("Stopping cron scheduler...")

	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop the cron scheduler - this waits for running jobs to complete
	ctx := s.cron.Stop()

	// Wait for all jobs to complete or timeout
	<-ctx.Done()

	log.Println("Cron scheduler stopped")
}

// AddRule dynamically adds a new automation rule to the scheduler
func (s *Scheduler) AddRule(rule domain.AutomationRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate cron expression
	if err := s.validateCronExpression(rule.CronSchedule); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Check if rule already exists
	if _, exists := s.entryIDs[rule.ID]; exists {
		return fmt.Errorf("rule %s already exists in scheduler", rule.ID)
	}

	// Create job function that calls emailService.SendFollowUpByAutomation
	jobFunc := func() {
		s.executeAutomation(rule.ID)
	}

	// Add job to cron with rule.CronSchedule
	entryID, err := s.cron.AddFunc(rule.CronSchedule, jobFunc)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	// Store cron entry ID for removal
	s.entryIDs[rule.ID] = entryID

	return nil
}

// RemoveRule dynamically removes an automation rule from the scheduler
func (s *Scheduler) RemoveRule(automationID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Look up cron entry ID by automation ID
	entryID, exists := s.entryIDs[automationID]
	if !exists {
		log.Printf("Rule %s not found in scheduler", automationID)
		return
	}

	// Remove job from cron
	s.cron.Remove(entryID)

	// Clean up stored entry ID
	delete(s.entryIDs, automationID)

	log.Printf("Removed rule %s from scheduler", automationID)
}

// executeAutomation executes the automation rule and handles errors
func (s *Scheduler) executeAutomation(automationID string) {
	log.Printf("Executing automation %s", automationID)

	ctx := context.Background()

	if err := s.emailService.SendFollowUpByAutomation(ctx, automationID); err != nil {
		log.Printf("Error executing automation %s: %v", automationID, err)
		// In production, you might want to add retry logic or alerting
		return
	}

	log.Printf("Successfully executed automation %s", automationID)
}

// validateCronExpression validates a cron expression using robfig/cron parser
func (s *Scheduler) validateCronExpression(cronSchedule string) error {
	if cronSchedule == "" {
		return fmt.Errorf("cron schedule cannot be empty")
	}

	// Use robfig/cron parser to validate
	// The default parser accepts 5-part cron expressions (minute, hour, day of month, month, day of week)
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	_, err := parser.Parse(cronSchedule)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	return nil
}

// GetScheduledRuleCount returns the number of currently scheduled rules
func (s *Scheduler) GetScheduledRuleCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entryIDs)
}