package domain

// Config holds all application configuration
type Config struct {
	// Server
	Port int
	Env  string

	// Encryption
	AESSecretKey string
	JWTSecret    string

	// DragonflyDB
	DragonflyAddr     string
	DragonflyPassword string
	DragonflyDB       int

	// PostgreSQL
	PostgresDSN string

	// Jira OAuth
	JiraClientID     string
	JiraClientSecret string
	JiraRedirectURI  string
	JiraAuthBaseURL  string // for OAuth operations
	JiraAPIBaseURL   string

	// Email Provider
	IMAPHost string
	IMAPPort int
	SMTPHost string
	SMTPPort int

	// IMAP Polling
	IMAPPollIntervalSeconds int
}

// Validate checks if all required configuration fields are present and valid
func (c *Config) Validate() error {
	// Server validation
	if c.Port <= 0 || c.Port > 65535 {
		return &ValidationError{Field: "Port", Message: "must be between 1 and 65535"}
	}
	if c.Env == "" {
		return &ValidationError{Field: "Env", Message: "is required"}
	}

	// Encryption validation
	if c.AESSecretKey == "" {
		return &ValidationError{Field: "AESSecretKey", Message: "is required"}
	}
	if len(c.AESSecretKey) != 32 {
		return &ValidationError{Field: "AESSecretKey", Message: "must be exactly 32 characters"}
	}
	// DISABLED: JWT validation - JWT authentication removed
	// if c.JWTSecret == "" {
	//     return &ValidationError{Field: "JWTSecret", Message: "is required"}
	// }

	// DragonflyDB validation
	if c.DragonflyAddr == "" {
		return &ValidationError{Field: "DragonflyAddr", Message: "is required"}
	}
	if c.DragonflyDB < 0 || c.DragonflyDB > 15 {
		return &ValidationError{Field: "DragonflyDB", Message: "must be between 0 and 15"}
	}

	// PostgreSQL validation
	if c.PostgresDSN == "" {
		return &ValidationError{Field: "PostgresDSN", Message: "is required"}
	}

	// Jira OAuth validation
	if c.JiraClientID == "" {
		return &ValidationError{Field: "JiraClientID", Message: "is required"}
	}
	if c.JiraClientSecret == "" {
		return &ValidationError{Field: "JiraClientSecret", Message: "is required"}
	}
	if c.JiraRedirectURI == "" {
		return &ValidationError{Field: "JiraRedirectURI", Message: "is required"}
	}
	if c.JiraAuthBaseURL == "" {
		return &ValidationError{Field: "JiraAuthBaseURL", Message: "is required"}
	}
	if c.JiraAPIBaseURL == "" {
		return &ValidationError{Field: "JiraAPIBaseURL", Message: "is required"}
	}

	// Email Provider validation
	if c.IMAPHost == "" {
		return &ValidationError{Field: "IMAPHost", Message: "is required"}
	}
	if c.IMAPPort <= 0 || c.IMAPPort > 65535 {
		return &ValidationError{Field: "IMAPPort", Message: "must be between 1 and 65535"}
	}
	if c.SMTPHost == "" {
		return &ValidationError{Field: "SMTPHost", Message: "is required"}
	}
	if c.SMTPPort <= 0 || c.SMTPPort > 65535 {
		return &ValidationError{Field: "SMTPPort", Message: "must be between 1 and 65535"}
	}

	// IMAP Polling validation
	if c.IMAPPollIntervalSeconds <= 0 {
		return &ValidationError{Field: "IMAPPollIntervalSeconds", Message: "must be greater than 0"}
	}

	return nil
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + " " + e.Message
}
