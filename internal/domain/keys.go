package domain

// DragonflyDB storage key conventions
const (
	// User keys
	UserKeyPrefix = "user:"
	UserEmailIndexPrefix = "user:index:email:"

	// OAuth token keys
	OAuthKeyPrefix = "oauth:"

	// Email credential keys
	EmailCredentialKeyPrefix = "email_credential:"

	// Automation rule keys
	AutomationKeyPrefix = "automation:"
	AutomationIndexPrefix = "automation:index:"

	// Email thread keys
	EmailThreadKeyPrefix = "email_thread:"
	EmailThreadIndexPrefix = "email_thread:index:"
)

// Key generators
func UserKey(id string) string {
	return UserKeyPrefix + id
}

func UserEmailIndexKey(email string) string {
	return UserEmailIndexPrefix + email
}

func OAuthKey(userID, provider string) string {
	return OAuthKeyPrefix + userID + ":" + provider
}

func EmailCredentialKey(userID string) string {
	return EmailCredentialKeyPrefix + userID
}

func AutomationKey(id string) string {
	return AutomationKeyPrefix + id
}

func AutomationIndexKey(userID string) string {
	return AutomationIndexPrefix + userID
}

func EmailThreadKey(id string) string {
	return EmailThreadKeyPrefix + id
}

func EmailThreadIndexKey(automationID string) string {
	return EmailThreadIndexPrefix + automationID
}
