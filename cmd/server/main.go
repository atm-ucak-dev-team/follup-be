package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/atm-ucak/follup/config"
	"github.com/atm-ucak/follup/internal/cron"
	"github.com/atm-ucak/follup/internal/email"
	"github.com/atm-ucak/follup/internal/handler"
	"github.com/atm-ucak/follup/internal/infra"
	"github.com/atm-ucak/follup/internal/repository"
	"github.com/atm-ucak/follup/internal/repository/dragonfly"
	"github.com/atm-ucak/follup/internal/repository/postgres"
	"github.com/atm-ucak/follup/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func dummyAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.Request().Header.Get("X-User-Dummy-Id")
		if userID == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "missing X-User-Dummy-Id header",
			})
		}
		c.Set("user_id", userID)
		return next(c)
	}
}

func main() {
	// 1. Load config (Viper)
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	log.Printf("Starting Jira Email Automation Backend in %s mode", cfg.Env)
	log.Printf("Server port: %d", cfg.Port)

	// 2. Init infra
	// Initialize DragonflyDB client
	dragonflyClient := infra.NewDragonflyClient(cfg)

	// Test DragonflyDB connection
	ctx := context.Background()
	if err := infra.Ping(ctx, dragonflyClient); err != nil {
		log.Fatal("Failed to connect to DragonflyDB:", err)
	}
	log.Println("Connected to DragonflyDB")

	// Initialize PostgreSQL pool (for future use)
	postgresPool := infra.NewPostgresPool(cfg)
	if postgresPool != nil {
		log.Println("Connected to PostgreSQL")
		defer infra.ClosePostgresPool(postgresPool)

		// Run database migrations
		if err := infra.RunMigrations(cfg.PostgresDSN); err != nil {
			log.Fatalf("Failed to run database migrations: %v", err)
		}
	} else {
		log.Println("PostgreSQL not available (optional)")
	}

	// 3. Init repositories
	var userRepo repository.UserRepository
	var followupRepo repository.FollowupRepository
	var emailCredentialRepo repository.EmailCredentialRepository
	var oauthTokenRepo repository.OAuthTokenRepository
	var emailThreadRepo repository.EmailThreadRepository

	if postgresPool != nil {
		// Prefer PostgreSQL implementations
		userRepo = postgres.NewUserRepository(postgresPool)
		followupRepo = postgres.NewFollowupRepository(postgresPool)
		emailCredentialRepo = postgres.NewEmailCredentialRepository(postgresPool)
		oauthTokenRepo = postgres.NewOAuthTokenRepository(postgresPool)
		emailThreadRepo = postgres.NewEmailThreadRepository(postgresPool)
		log.Println("Initialized PostgreSQL repositories")
	} else {
		// Fallback to DragonflyDB/Redis implementations
		userRepo = dragonfly.NewUserRepository(dragonflyClient)
		followupRepo = dragonfly.NewFollowupRepository(dragonflyClient)
		emailCredentialRepo = dragonfly.NewEmailCredentialRepository(dragonflyClient)
		oauthTokenRepo = dragonfly.NewOAuthTokenRepository(dragonflyClient)
		emailThreadRepo = dragonfly.NewEmailThreadRepository(dragonflyClient)
		log.Println("Initialized Dragonfly repositories")
	}

	// 4. Init services (inject repos)
	authService := service.NewAuthService(userRepo, oauthTokenRepo, followupRepo, cfg)
	jiraService := service.NewJiraService(cfg)
	emailService := service.NewEmailService(emailCredentialRepo, followupRepo, emailThreadRepo, cfg)
	automationService := service.NewAutomationService(followupRepo, emailThreadRepo, emailService)

	log.Println("Initialized services")

	// 5. Init handlers (inject services)
	authHandler := handler.NewAuthHandler(authService, cfg)
	jiraHandler := handler.NewJiraHandler(jiraService)
	emailHandler := handler.NewEmailHandler(emailService)
	automationHandler := handler.NewAutomationHandler(automationService)
	followupHandler := handler.NewFollowupHandler(automationService)
	ticketHandler := handler.NewTicketHandler(jiraService, automationService)

	log.Println("Initialized handlers")

	// 6. Init cron scheduler (inject services)
	scheduler := cron.NewScheduler(followupRepo, emailService, emailThreadRepo)
	if err := scheduler.Start(); err != nil {
		log.Fatal("Failed to start cron scheduler:", err)
	}
	log.Println("Started cron scheduler")

	// 7. Init IMAP poller goroutine
	pollInterval := time.Duration(cfg.IMAPPollIntervalSeconds) * time.Second
	poller := email.NewPoller(emailService, followupRepo, emailThreadRepo, pollInterval)
	poller.Start()
	log.Printf("Started IMAP poller with %v interval", pollInterval)

	// 8. Register Echo routes
	e := echo.New()

	// Middleware
	// e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Health check endpoint (no auth required)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Auth routes (no JWT required)
	e.GET("/auth/jira/connect", authHandler.ConnectJira)
	e.GET("/auth/jira/callback", authHandler.JiraCallback)
	e.POST("/auth/jira/refresh", authHandler.RefreshToken)
	// Dummy callback endpoint for frontend testing (development only)
	e.GET("/auth/jira/dummy-callback", authHandler.DummyJiraCallback)

	// Dev token endpoint (development only)
	e.GET("/auth/dev/token", authHandler.DevToken)

	// Protected routes (auth via X-User-Dummy-Id header)
	api := e.Group("/api/v1")
	api.Use(dummyAuthMiddleware)

	// Email credentials routes
	api.POST("/email/credentials", emailHandler.SaveCredentials)
	api.GET("/email/credentials", emailHandler.GetCredentials)

	// Jira routes
	api.GET("/jira/issues", jiraHandler.GetIssues)
	api.GET("/jira/issues/:ticket_key", jiraHandler.GetIssue)

	// Automation routes
	api.POST("/automations", automationHandler.CreateAutomation)
	api.GET("/automations", automationHandler.ListAutomations)
	api.GET("/automations/:id", automationHandler.GetAutomation)
	api.PATCH("/automations/:id", automationHandler.UpdateAutomation)
	api.DELETE("/automations/:id", automationHandler.DeleteAutomation)
	api.POST("/automations/:id/trigger", automationHandler.TriggerAutomation)

	// Followup routes
	api.POST("/followups", followupHandler.CreateFollowup)
	api.GET("/followup", followupHandler.ListFollowups)
	api.GET("/:jiraTicketID/followups", followupHandler.GetFollowupsByTicketID)
	api.GET("/:jiraTicketID/summary", followupHandler.GetSummary)
	api.GET("/statistic", followupHandler.GetGlobalSummary)

	// Ticket routes
	api.GET("/tickets", ticketHandler.GetTickets)

	log.Println("Registered HTTP routes")

	// 9. Start Echo server in background goroutine
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		log.Printf("Starting HTTP server on %s", addr)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 10. Handle graceful shutdown
	// Wait for interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Received shutdown signal, gracefully shutting down...")

	// Cleanup in reverse order of initialization
	// 10.1. Shutdown HTTP server with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during HTTP server shutdown: %v", err)
	}
	log.Println("HTTP server stopped")

	// 10.2. Stop IMAP poller
	poller.Stop()
	log.Println("IMAP poller stopped")

	// 10.3. Stop cron scheduler
	scheduler.Stop()
	log.Println("Cron scheduler stopped")

	// 10.4. Close database connections
	if err := infra.Close(dragonflyClient); err != nil {
		log.Printf("Error closing DragonflyDB: %v", err)
	}
	log.Println("DragonflyDB connection closed")

	log.Println("Application shutdown complete")
}
