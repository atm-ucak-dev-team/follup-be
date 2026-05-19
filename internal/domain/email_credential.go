package domain

import "time"

// EmailCredential represents email credentials for IMAP/SMTP
type EmailCredential struct {
	UserID            string    `json:"user_id"`
	EmailAddress      string    `json:"email_address"`
	EncryptedPassword string    `json:"encrypted_password"` // AES encrypted
	IMAPHost          string    `json:"imap_host"`          // hardcoded at app level
	SMTPHost          string    `json:"smtp_host"`          // hardcoded at app level
	CreatedAt         time.Time `json:"created_at"`
}
