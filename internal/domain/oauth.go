package domain

import "time"

// OAuthToken represents an OAuth token for a provider (e.g., Jira)
type OAuthToken struct {
	UserID       string    `json:"user_id"`
	Provider     string    `json:"provider"` // e.g., "jira"
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}
