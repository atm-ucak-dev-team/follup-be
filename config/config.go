package config

import (
	"fmt"
	"strings"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/spf13/viper"
)

// LoadConfig loads configuration from .env file and environment variables
func LoadConfig() (*domain.Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../../..")
	viper.AddConfigPath("../../../config")
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Read .env file (if it exists)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// .env file not found, that's ok - rely on environment variables
	}

	// Bind environment variables
	bindEnvVars()

	// Build config
	cfg, err := buildConfig()
	if err != nil {
		return nil, fmt.Errorf("error building config: %w", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// setDefaults sets default values for optional configuration fields
func setDefaults() {
	// Server defaults
	viper.SetDefault("APP_PORT", 8080)
	viper.SetDefault("APP_ENV", "development")

	// DragonflyDB defaults
	viper.SetDefault("DRAGONFLY_ADDR", "localhost:6379")
	viper.SetDefault("DRAGONFLY_DB", 0)

	// Email Provider defaults
	viper.SetDefault("IMAP_HOST", "imap.gmail.com")
	viper.SetDefault("IMAP_PORT", 993)
	viper.SetDefault("SMTP_HOST", "smtp.gmail.com")
	viper.SetDefault("SMTP_PORT", 587)

	// Frontend callback URL defaults
	viper.SetDefault("FRONTEND_CALLBACK_URL", "follupapp://callback")

	// IMAP Polling defaults
	viper.SetDefault("IMAP_POLL_INTERVAL_SECONDS", 300)

	// Seed defaults
	viper.SetDefault("RUN_SEED", false)
}

// bindEnvVars binds environment variables to viper keys
func bindEnvVars() {
	// Server
	viper.BindEnv("APP_PORT", "APP_PORT")
	viper.BindEnv("APP_ENV", "APP_ENV")

	// Encryption
	viper.BindEnv("AES_SECRET_KEY", "AES_SECRET_KEY")
	viper.BindEnv("JWT_SECRET", "JWT_SECRET")

	// DragonflyDB
	viper.BindEnv("DRAGONFLY_ADDR", "DRAGONFLY_ADDR")
	viper.BindEnv("DRAGONFLY_PASSWORD", "DRAGONFLY_PASSWORD")
	viper.BindEnv("DRAGONFLY_DB", "DRAGONFLY_DB")

	// PostgreSQL
	viper.BindEnv("POSTGRES_DSN", "POSTGRES_DSN")

	// Jira OAuth
	viper.BindEnv("JIRA_CLIENT_ID", "JIRA_CLIENT_ID")
	viper.BindEnv("JIRA_CLIENT_SECRET", "JIRA_CLIENT_SECRET")
	viper.BindEnv("JIRA_REDIRECT_URI", "JIRA_REDIRECT_URI")
	viper.BindEnv("JIRA_AUTH_BASE_URL", "JIRA_AUTH_BASE_URL")
	viper.BindEnv("JIRA_API_BASE_URL", "JIRA_API_BASE_URL")
	viper.BindEnv("FRONTEND_CALLBACK_URL", "FRONTEND_CALLBACK_URL")

	// Email Provider
	viper.BindEnv("IMAP_HOST", "IMAP_HOST")
	viper.BindEnv("IMAP_PORT", "IMAP_PORT")
	viper.BindEnv("SMTP_HOST", "SMTP_HOST")
	viper.BindEnv("SMTP_PORT", "SMTP_PORT")

	// IMAP Polling
	viper.BindEnv("IMAP_POLL_INTERVAL_SECONDS", "IMAP_POLL_INTERVAL_SECONDS")

	// Seed
	viper.BindEnv("RUN_SEED", "RUN_SEED")
}

// buildConfig constructs the Config struct from viper values
func buildConfig() (*domain.Config, error) {
	return &domain.Config{
		// Server
		Port: viper.GetInt("APP_PORT"),
		Env:  viper.GetString("APP_ENV"),

		// Encryption
		AESSecretKey: viper.GetString("AES_SECRET_KEY"),
		JWTSecret:    viper.GetString("JWT_SECRET"),

		// DragonflyDB
		DragonflyAddr:     viper.GetString("DRAGONFLY_ADDR"),
		DragonflyPassword: viper.GetString("DRAGONFLY_PASSWORD"),
		DragonflyDB:       viper.GetInt("DRAGONFLY_DB"),

		// PostgreSQL
		PostgresDSN: viper.GetString("POSTGRES_DSN"),

		// Jira OAuth
		JiraClientID:        viper.GetString("JIRA_CLIENT_ID"),
		JiraClientSecret:    viper.GetString("JIRA_CLIENT_SECRET"),
		JiraRedirectURI:     viper.GetString("JIRA_REDIRECT_URI"),
		JiraAuthBaseURL:     viper.GetString("JIRA_AUTH_BASE_URL"),
		JiraAPIBaseURL:      viper.GetString("JIRA_API_BASE_URL"),
		FrontendCallbackURL: viper.GetString("FRONTEND_CALLBACK_URL"),

		// Email Provider
		IMAPHost: viper.GetString("IMAP_HOST"),
		IMAPPort: viper.GetInt("IMAP_PORT"),
		SMTPHost: viper.GetString("SMTP_HOST"),
		SMTPPort: viper.GetInt("SMTP_PORT"),

		// IMAP Polling
		IMAPPollIntervalSeconds: viper.GetInt("IMAP_POLL_INTERVAL_SECONDS"),

		// Seed
		RunSeed: viper.GetBool("RUN_SEED"),
	}, nil
}

// IsProduction checks if the application is running in production mode
func IsProduction(cfg *domain.Config) bool {
	return strings.ToLower(cfg.Env) == "production"
}

// IsDevelopment checks if the application is running in development mode
func IsDevelopment(cfg *domain.Config) bool {
	return strings.ToLower(cfg.Env) == "development"
}
