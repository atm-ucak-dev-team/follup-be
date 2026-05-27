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

	// Fetch envelope headers for proper thread matching
	messages := make(chan *imap.Message, len(uids))
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
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
	var messageID, inReplyTo, references string
	if msg.Envelope != nil {
		messageID = msg.Envelope.MessageId
		// InReplyTo is a slice of message IDs
		if len(msg.Envelope.InReplyTo) > 0 {
			inReplyTo = string(msg.Envelope.InReplyTo[0])
		}
		// References header may not be directly available in Envelope
		// We'll need to handle this case in the IMAP processing
	}

	if messageID == "" {
		return fmt.Errorf("message has no Message-ID")
	}

	// Try to find matching thread
	threadID, err := p.findMatchingThread(ctx, userID, messageID, inReplyTo, references)
	if err != nil {
		return err
	}

	if threadID == "" {
		log.Printf("No matching thread found for message %s", messageID)
		return nil
	}

	// Update email thread status to 'replied'
	if err := p.emailThreadRepo.UpdateThreadStatus(ctx, threadID, domain.EmailThreadStatusReplied); err != nil {
		return fmt.Errorf("failed to update thread status: %w", err)
	}

	// Get thread details to find associated followup
	thread, err := p.emailThreadRepo.GetByID(ctx, threadID)
	if err != nil {
		return fmt.Errorf("failed to get thread: %w", err)
	}

	// Update followup status to 'replied'
	followup, err := p.followupRepo.GetByID(ctx, thread.AutomationID)
	if err != nil {
		return fmt.Errorf("failed to get followup: %w", err)
	}

	// Only update if the followup is still ongoing
	if followup.Status == domain.FollowupStatusOngoing {
		followup.Status = domain.FollowupStatusReplied
		if err := p.followupRepo.Update(ctx, followup); err != nil {
			return fmt.Errorf("failed to update followup status: %w", err)
		}
		log.Printf("Updated followup %s status to 'replied'", followup.ID)
	}

	log.Printf("Updated thread %s status to 'replied' for message %s", threadID, messageID)
	return nil
}

// findMatchingThread finds a thread that matches the given message using In-Reply-To and References headers
func (p *Poller) findMatchingThread(ctx context.Context, userID, messageID, inReplyTo, references string) (string, error) {
	// Try to match by In-Reply-To header first
	if inReplyTo != "" {
		thread, err := p.emailThreadRepo.GetByGmailThreadID(ctx, inReplyTo)
		if err == nil && thread != nil && thread.UserID == userID {
			return thread.ID, nil
		}
	}

	// Try to match by References header
	if references != "" {
		// Split by spaces to handle multiple message IDs in References
		refIDs := splitReferences(references)
		for _, refID := range refIDs {
			thread, err := p.emailThreadRepo.GetByGmailThreadID(ctx, refID)
			if err == nil && thread != nil && thread.UserID == userID {
				return thread.ID, nil
			}
		}
	}

	// Try to match by Message-ID (in case this is a direct reply)
	if messageID != "" {
		thread, err := p.emailThreadRepo.GetByGmailThreadID(ctx, messageID)
		if err == nil && thread != nil && thread.UserID == userID {
			return thread.ID, nil
		}
	}

	return "", nil
}

// splitReferences splits the References header by spaces and handles angle brackets
func splitReferences(references string) []string {
	var ids []string
	currentID := ""
	inAngleBracket := false

	for _, char := range references {
		switch char {
		case '<':
			inAngleBracket = true
			currentID = ""
		case '>':
			inAngleBracket = false
			if currentID != "" {
				ids = append(ids, currentID)
			}
		case ' ', '\t', '\n', '\r':
			if inAngleBracket {
				currentID += string(char)
			}
		default:
			if inAngleBracket {
				currentID += string(char)
			}
		}
	}

	return ids
}
