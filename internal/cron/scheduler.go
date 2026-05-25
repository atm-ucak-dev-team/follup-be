package cron

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/atm-ucak/follup/internal/repository"
	"github.com/atm-ucak/follup/internal/service"
	"github.com/robfig/cron/v3"
)

// Scheduler manages followup rule execution using cron
type Scheduler struct {
	cron         *cron.Cron
	followupRepo repository.FollowupRepository
	emailService service.EmailService
	entryIDs     map[string]cron.EntryID
	mu           sync.RWMutex
}

// NewScheduler creates a new cron scheduler instance
func NewScheduler(followupRepo repository.FollowupRepository, emailService service.EmailService) *Scheduler {
	return &Scheduler{
		cron:         cron.New(),
		followupRepo: followupRepo,
		emailService: emailService,
		entryIDs:     make(map[string]cron.EntryID),
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

	jobFunc := func() {
		s.executeAutomation(rule.ID)
	}

	entryID, err := s.cron.AddFunc(rule.Frequency, jobFunc)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.entryIDs[rule.ID] = entryID

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

	if err := s.emailService.SendFollowUpByAutomation(ctx, automationID); err != nil {
		log.Printf("Error executing followup %s: %v", automationID, err)
		return
	}

	log.Printf("Successfully executed followup %s", automationID)
}

// GetScheduledRuleCount returns the number of currently scheduled rules
func (s *Scheduler) GetScheduledRuleCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entryIDs)
}
