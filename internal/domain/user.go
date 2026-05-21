package domain

import "time"

// User represents a user in the system
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CloudID   string    `json:"cloud_id"`   // Jira Cloud instance ID
	AvatarURL string    `json:"avatar_url"` // User avatar URL (32x32)
	CreatedAt time.Time `json:"created_at"`
}
