package config

import (
	"errors"
	"os"
	"testing"

	"github.com/atm-ucak/follup/internal/domain"
)

func TestLoadConfig_Success(t *testing.T) {
	// Set up environment variables
	setEnvVars(t)
	defer unsetEnvVars(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Verify critical fields
	if cfg.Port != 8080 {
		t.Errorf("Expected Port 8080, got %d", cfg.Port)
	}
	if cfg.Env != "development" {
		t.Errorf("Expected Env 'development', got '%s'", cfg.Env)
	}
	if cfg.AESSecretKey != "12345678901234567890123456789012" {
		t.Errorf("Expected AES key '12345678901234567890123456789012', got '%s'", cfg.AESSecretKey)
	}
	if cfg.DragonflyAddr != "localhost:6379" {
		t.Errorf("Expected DragonflyAddr 'localhost:6379', got '%s'", cfg.DragonflyAddr)
	}
	if cfg.IMAPHost != "imap.gmail.com" {
		t.Errorf("Expected IMAPHost 'imap.gmail.com', got '%s'", cfg.IMAPHost)
	}
	if cfg.SMTPHost != "smtp.gmail.com" {
		t.Errorf("Expected SMTPHost 'smtp.gmail.com', got '%s'", cfg.SMTPHost)
	}
}

func TestLoadConfig_MissingRequiredField(t *testing.T) {
	// Set up environment variables but miss one
	envVars := map[string]string{
		"APP_PORT":                      "8080",
		"APP_ENV":                       "development",
		"AES_SECRET_KEY":                "12345678901234567890123456789012", // exactly 32 chars
		"JWT_SECRET":                    "test-jwt-secret-key-exactly-32-chars!!", // exactly 32 chars
		"DRAGONFLY_ADDR":                "localhost:6379",
		"DRAGONFLY_DB":                  "0",
		"POSTGRES_DSN":                  "postgres://user:pass@localhost:5432/dbname",
		// Missing JIRA_CLIENT_ID
		"JIRA_CLIENT_SECRET":            "secret",
		"JIRA_REDIRECT_URI":             "https://example.com/callback",
		"JIRA_BASE_URL":                 "https://api.atlassian.com",
		"IMAP_HOST":                     "imap.gmail.com",
		"IMAP_PORT":                     "993",
		"SMTP_HOST":                     "smtp.gmail.com",
		"SMTP_PORT":                     "587",
		"IMAP_POLL_INTERVAL_SECONDS":    "300",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}
	defer unsetEnvVars(t)

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error for missing JIRA_CLIENT_ID, got nil")
	}

	var validationErr *domain.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Expected ValidationError, got %T", err)
	}
	if validationErr.Field != "JiraClientID" {
		t.Errorf("Expected error on field 'JiraClientID', got '%s'", validationErr.Field)
	}
}

func TestLoadConfig_InvalidPortNumber(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		expectError bool
		errorField  string
	}{
		{"Valid port", "8080", false, ""},
		{"Port too high", "70000", true, "Port"},
		{"Port zero", "0", true, "Port"},
		{"Port negative", "-1", true, "Port"},
		{"Port max valid", "65535", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvVars(t)
			defer unsetEnvVars(t)

			os.Setenv("APP_PORT", tt.port)

			cfg, err := LoadConfig()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error for invalid port, got nil")
				}
				var validationErr *domain.ValidationError
				if !errors.As(err, &validationErr) {
					t.Fatalf("Expected ValidationError, got %T", err)
				}
				if validationErr.Field != tt.errorField {
					t.Errorf("Expected error on field '%s', got '%s'", tt.errorField, validationErr.Field)
				}
			} else {
				if err != nil {
					t.Errorf("LoadConfig() failed: %v", err)
				}
				if cfg == nil {
					t.Fatal("Expected config, got nil")
				}
				// For valid port tests, check that the port matches what was set
				if tt.name == "Valid port" && cfg.Port != 8080 {
					t.Errorf("Expected Port 8080, got %d", cfg.Port)
				}
				if tt.name == "Port max valid" && cfg.Port != 65535 {
					t.Errorf("Expected Port 65535, got %d", cfg.Port)
				}
			}
		})
	}
}

func TestLoadConfig_InvalidInterval(t *testing.T) {
	setEnvVars(t)
	defer unsetEnvVars(t)

	// Set invalid interval (zero)
	os.Setenv("IMAP_POLL_INTERVAL_SECONDS", "0")

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error for zero interval, got nil")
	}

	var validationErr *domain.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Expected ValidationError, got %T", err)
	}
	if validationErr.Field != "IMAPPollIntervalSeconds" {
		t.Errorf("Expected error on field 'IMAPPollIntervalSeconds', got '%s'", validationErr.Field)
	}
}

func TestLoadConfig_InvalidAESKey(t *testing.T) {
	tests := []struct {
		name        string
		aesKey      string
		expectError bool
	}{
		{"Valid 32 char key", "12345678901234567890123456789012", false},
		{"Too short", "short-key", true},
		{"Too long", "this-key-is-way-too-long-for-32-characters", true},
		{"Exactly 32 chars", "12345678901234567890123456789012", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvVars(t)
			defer unsetEnvVars(t)

			os.Setenv("AES_SECRET_KEY", tt.aesKey)

			_, err := LoadConfig()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error for invalid AES key, got nil")
				}
				var validationErr *domain.ValidationError
				if !errors.As(err, &validationErr) {
					t.Fatalf("Expected ValidationError, got %T", err)
				}
				if validationErr.Field != "AESSecretKey" {
					t.Errorf("Expected error on field 'AESSecretKey', got '%s'", validationErr.Field)
				}
			} else {
				if err != nil {
					t.Errorf("LoadConfig() failed: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_DragonflyDB(t *testing.T) {
	tests := []struct {
		name        string
		dbNum       int
		expectError bool
	}{
		{"Valid DB 0", 0, false},
		{"Valid DB 15", 15, false},
		{"Invalid DB -1", -1, true},
		{"Invalid DB 16", 16, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &domain.Config{
				Port:                    8080,
				Env:                     "development",
				AESSecretKey:            "12345678901234567890123456789012", // exactly 32 chars
				JWTSecret:               "test-jwt-secret-key-exactly-32-chars!!", // exactly 32 chars
				DragonflyAddr:           "localhost:6379",
				DragonflyPassword:       "",
				DragonflyDB:             tt.dbNum,
				PostgresDSN:             "postgres://user:pass@localhost:5432/dbname",
				JiraClientID:            "client-id",
				JiraClientSecret:        "client-secret",
				JiraRedirectURI:         "https://example.com/callback",
				JiraBaseURL:             "https://api.atlassian.com",
				IMAPHost:                "imap.gmail.com",
				IMAPPort:                993,
				SMTPHost:                "smtp.gmail.com",
				SMTPPort:                587,
				IMAPPollIntervalSeconds: 300,
			}

			err := cfg.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error for invalid DragonflyDB, got nil")
				}
				var validationErr *domain.ValidationError
				if !errors.As(err, &validationErr) {
					t.Fatalf("Expected ValidationError, got %T", err)
				}
				if validationErr.Field != "DragonflyDB" {
					t.Errorf("Expected error on field 'DragonflyDB', got '%s'", validationErr.Field)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() failed: %v", err)
				}
			}
		})
	}
}

func TestIsProduction(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected bool
	}{
		{"Production", "production", true},
		{"Production uppercase", "PRODUCTION", true},
		{"Development", "development", false},
		{"Staging", "staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &domain.Config{Env: tt.env}
			result := IsProduction(cfg)
			if result != tt.expected {
				t.Errorf("IsProduction() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsDevelopment(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected bool
	}{
		{"Development", "development", true},
		{"Development uppercase", "DEVELOPMENT", true},
		{"Production", "production", false},
		{"Staging", "staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &domain.Config{Env: tt.env}
			result := IsDevelopment(cfg)
			if result != tt.expected {
				t.Errorf("IsDevelopment() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Helper functions

func setEnvVars(t *testing.T) {
	envVars := map[string]string{
		"APP_PORT":                   "8080",
		"APP_ENV":                    "development",
		"AES_SECRET_KEY":             "12345678901234567890123456789012", // exactly 32 chars
		"JWT_SECRET":                 "test-jwt-secret-key-exactly-32-chars!!", // exactly 32 chars
		"DRAGONFLY_ADDR":             "localhost:6379",
		"DRAGONFLY_PASSWORD":         "",
		"DRAGONFLY_DB":               "0",
		"POSTGRES_DSN":               "postgres://user:pass@localhost:5432/dbname",
		"JIRA_CLIENT_ID":             "test-client-id",
		"JIRA_CLIENT_SECRET":         "test-client-secret",
		"JIRA_REDIRECT_URI":          "https://example.com/callback",
		"JIRA_BASE_URL":              "https://api.atlassian.com",
		"IMAP_HOST":                  "imap.gmail.com",
		"IMAP_PORT":                  "993",
		"SMTP_HOST":                  "smtp.gmail.com",
		"SMTP_PORT":                  "587",
		"IMAP_POLL_INTERVAL_SECONDS": "300",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}
}

func unsetEnvVars(t *testing.T) {
	envVars := []string{
		"APP_PORT",
		"APP_ENV",
		"AES_SECRET_KEY",
		"JWT_SECRET",
		"DRAGONFLY_ADDR",
		"DRAGONFLY_PASSWORD",
		"DRAGONFLY_DB",
		"POSTGRES_DSN",
		"JIRA_CLIENT_ID",
		"JIRA_CLIENT_SECRET",
		"JIRA_REDIRECT_URI",
		"JIRA_BASE_URL",
		"IMAP_HOST",
		"IMAP_PORT",
		"SMTP_HOST",
		"SMTP_PORT",
		"IMAP_POLL_INTERVAL_SECONDS",
	}

	for _, v := range envVars {
		os.Unsetenv(v)
	}
}
