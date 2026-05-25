package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/atm-ucak/follup/internal/repository"
	"github.com/atm-ucak/follup/internal/service"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// Poller handles background IMAP polling for email replies
type Poller struct {
	emailService    service.EmailService
	followupRepo    repository.FollowupRepository
	emailThreadRepo repository.EmailThreadRepository
	interval        time.Duration
	stopChan        chan struct{}
	running         bool
}

// NewPoller creates a new Poller instance
func NewPoller(
	emailService service.EmailService,
	followupRepo repository.FollowupRepository,
	emailThreadRepo repository.EmailThreadRepository,
	interval time.Duration,
) *Poller {
	return &Poller{
		emailService:    emailService,
		followupRepo:    followupRepo,
		emailThreadRepo: emailThreadRepo,
		interval:        interval,
		stopChan:        make(chan struct{}),
		running:         false,
	}
}

// Start begins the background polling goroutine
func (p *Poller) Start() {
	if p.running {
		log.Println("Poller is already running")
		return
	}

	p.running = true
	log.Printf("Starting IMAP poller with %v interval", p.interval)

	go func() {
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()

		// Run immediately on start
		p.pollOnce()

		for {
			select {
			case <-ticker.C:
				p.pollOnce()
			case <-p.stopChan:
				log.Println("Poller stopped gracefully")
				return
			}
		}
	}()
}

// Stop gracefully stops the polling goroutine
func (p *Poller) Stop() {
	if !p.running {
		log.Println("Poller is not running")
		return
	}

	log.Println("Stopping IMAP poller...")
	p.running = false

	// Close stopChan safely (prevent double-close)
	select {
	case <-p.stopChan:
		// Channel already closed
	default:
		close(p.stopChan)
	}
}

// pollOnce performs a single polling cycle
func (p *Poller) pollOnce() {
	log.Println("Starting IMAP polling cycle...")

	ctx := context.Background()

	// Get all active followup rules
	activeRules, err := p.followupRepo.GetActiveRules(ctx)
	if err != nil {
		log.Printf("Failed to get active rules: %v", err)
		return
	}

	if len(activeRules) == 0 {
		log.Println("No active followups to poll")
		return
	}

	log.Printf("Found %d active followups to check", len(activeRules))

	// Group rules by user to avoid duplicate IMAP connections per user
	userRules := make(map[string][]*domain.Followup)
	for _, rule := range activeRules {
		userRules[rule.UserID] = append(userRules[rule.UserID], rule)
	}

	// Process each user's automations
	for userID, rules := range userRules {
		if err := p.pollUser(ctx, userID, rules); err != nil {
			log.Printf("Failed to poll user %s: %v", userID, err)
		}
	}

	log.Println("IMAP polling cycle completed")
}

// pollUser polls IMAP for a single user
func (p *Poller) pollUser(ctx context.Context, userID string, rules []*domain.Followup) error {
	// Get email credentials for the user
	cred, err := p.emailService.GetCredential(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Decrypt password
	password, err := p.emailService.DecryptPassword(cred.EncryptedPassword)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Connect to IMAP server
	imapClient, err := p.connectIMAP(cred.EmailAddress, password, cred.IMAPHost, 993)
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP: %w", err)
	}
	defer imapClient.Logout()

	// Select INBOX
	mbox, err := imapClient.Select("INBOX", false)
	if err != nil {
		return fmt.Errorf("failed to select INBOX: %w", err)
	}

	log.Printf("User %s: Connected to IMAP, %d messages in INBOX", userID, mbox.Messages)

	// Get all threads for this user's automations to match against
	var threadIDs []string
	for _, rule := range rules {
		threadIDs = append(threadIDs, rule.ID)
	}

	// Search for unseen messages
	searchCriteria := imap.NewSearchCriteria()
	searchCriteria.WithoutFlags = []string{imap.SeenFlag}

	uids, err := imapClient.Search(searchCriteria)
	if err != nil {
		return fmt.Errorf("failed to search messages: %w", err)
	}

	if len(uids) == 0 {
		log.Printf("User %s: No unseen messages found", userID)
		return nil
	}

	log.Printf("User %s: Found %d unseen messages", userID, len(uids))

	// Fetch message headers for thread matching
	if err := p.fetchAndMatchMessages(ctx, imapClient, uids, userID); err != nil {
		return fmt.Errorf("failed to fetch and match messages: %w", err)
	}

	return nil
}

// connectIMAP establishes a TLS connection to IMAP server
func (p *Poller) connectIMAP(email, password, host string, port int) (*client.Client, error) {
	server := fmt.Sprintf("%s:%d", host, port)

	// Create TLS configuration
	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: false, // Proper TLS verification
	}

	// Connect with TLS
	imapClient, err := client.DialTLS(server, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	// Login
	if err := imapClient.Login(email, password); err != nil {
		imapClient.Logout()
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	return imapClient, nil
}

// fetchAndMatchMessages fetches message headers and matches them to threads
func (p *Poller) fetchAndMatchMessages(ctx context.Context, imapClient *client.Client, uids []uint32, userID string) error {
	// Create sequence set for all UIDs
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	// Fetch only the headers we need for matching
	messages := make(chan *imap.Message, len(uids))
	items := []imap.FetchItem{
		imap.FetchBody,
		imap.FetchEnvelope,
	}

	if err := imapClient.Fetch(seqset, items, messages); err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	// Process each message
	for msg := range messages {
		if msg == nil {
			continue
		}

		if err := p.matchMessageToThread(ctx, msg, userID); err != nil {
			log.Printf("Failed to match message: %v", err)
			continue
		}
	}

	return nil
}

// matchMessageToThread matches a message to existing threads and updates status
func (p *Poller) matchMessageToThread(ctx context.Context, msg *imap.Message, userID string) error {
	// Extract message headers
	var messageID string
	if msg.Envelope != nil {
		messageID = msg.Envelope.MessageId
	}

	if messageID == "" {
		return fmt.Errorf("message has no Message-ID")
	}

	// Try to find matching thread
	threadID, err := p.findMatchingThread(ctx, userID, messageID)
	if err != nil {
		return err
	}

	if threadID == "" {
		log.Printf("No matching thread found for message %s", messageID)
		return nil
	}

	// Update thread status to 'replied'
	if err := p.emailThreadRepo.UpdateThreadStatus(ctx, threadID, domain.EmailThreadStatusReplied); err != nil {
		return fmt.Errorf("failed to update thread status: %w", err)
	}

	log.Printf("Updated thread %s status to 'replied' for message %s", threadID, messageID)
	return nil
}

// findMatchingThread finds a thread that matches the given message
func (p *Poller) findMatchingThread(ctx context.Context, userID, messageID string) (string, error) {
	// Get all threads for the user
	// This is a simplified implementation - in production, you'd want a more efficient matching algorithm
	// that checks In-Reply-To and References headers

	// For now, we'll implement a basic matching strategy
	// In a real implementation, you would:
	// 1. Check if messageID is in References field of any thread
	// 2. Check if messageID is In-Reply-To any thread's message
	// 3. Check thread lineage

	// This is a placeholder that would need to be implemented based on your thread storage strategy
	return "", nil
}
