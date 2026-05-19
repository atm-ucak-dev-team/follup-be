package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	// "github.com/golang-jwt/jwt/v5" // DISABLED: JWT authentication
	"github.com/bomanarakasura/jira-email-automation/internal/domain"
	"github.com/bomanarakasura/jira-email-automation/internal/repository"
)

// AuthServiceImpl implements the AuthService interface
type AuthServiceImpl struct {
	userRepo       repository.UserRepository
	oauthRepo      repository.OAuthTokenRepository
	automationRepo repository.AutomationRuleRepository
	config         *domain.Config
	httpClient     *http.Client
}

// NewAuthService creates a new AuthService instance
func NewAuthService(
	userRepo repository.UserRepository,
	oauthRepo repository.OAuthTokenRepository,
	automationRepo repository.AutomationRuleRepository,
	config *domain.Config,
) AuthService {
	return &AuthServiceImpl{
		userRepo:       userRepo,
		oauthRepo:      oauthRepo,
		automationRepo: automationRepo,
		config:         config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Jira token response from OAuth endpoint
type jiraTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

// Jira user info response
type jiraUserResponse struct {
	AccountID string `json:"account_id"`
	Name      string `json:"name"`
	Email     string `json:"email_address"`
}

// ExchangeJiraCode exchanges the authorization code for access token and returns user with JWT
func (s *AuthServiceImpl) ExchangeJiraCode(ctx context.Context, code, state string) (*domain.User, string, error) {
	// Exchange code for token
	tokenResp, err := s.exchangeCodeForToken(ctx, code)
	if err != nil {
		return nil, "", fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user info from Jira
	jiraUser, err := s.getJiraUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get Jira user info: %w", err)
	}

	// Get or create user
	user, err := s.getOrCreateUser(ctx, jiraUser)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get or create user: %w", err)
	}

	// Save OAuth token with TTL
	oauthToken := &domain.OAuthToken{
		UserID:       user.ID,
		Provider:     "jira",
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	if err := s.oauthRepo.Create(ctx, oauthToken); err != nil {
		return nil, "", fmt.Errorf("failed to save OAuth token: %w", err)
	}

	// DISABLED: JWT generation
	// jwtToken, err := s.generateJWT(user.ID)
	// if err != nil {
	//     return nil, "", fmt.Errorf("failed to generate JWT: %w", err)
	// }

	// Return empty token instead of JWT
	jwtToken := ""

	return user, jwtToken, nil
}

// RefreshJiraToken refreshes an expired access token and returns new access token
func (s *AuthServiceImpl) RefreshJiraToken(ctx context.Context, userID string) (string, error) {
	// Get existing OAuth token
	oauthToken, err := s.oauthRepo.GetByUserIDAndProvider(ctx, userID, "jira")
	if err != nil {
		return "", fmt.Errorf("failed to get OAuth token: %w", err)
	}

	// Exchange refresh token for new access token
	tokenResp, err := s.refreshToken(ctx, oauthToken.RefreshToken)
	if err != nil {
		// Pause all user automations on refresh failure
		if pauseErr := s.pauseUserAutomations(ctx, userID); pauseErr != nil {
			return "", fmt.Errorf("failed to refresh token: %v, failed to pause automations: %w", err, pauseErr)
		}
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update token in repository
	oauthToken.AccessToken = tokenResp.AccessToken
	oauthToken.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	if tokenResp.RefreshToken != "" {
		oauthToken.RefreshToken = tokenResp.RefreshToken
	}

	if err := s.oauthRepo.Update(ctx, oauthToken); err != nil {
		return "", fmt.Errorf("failed to update OAuth token: %w", err)
	}

	return oauthToken.AccessToken, nil
}

// GenerateAuthURL creates the Jira OAuth authorization URL
func (s *AuthServiceImpl) GenerateAuthURL(state string) string {
	params := url.Values{}
	params.Add("client_id", s.config.JiraClientID)
	params.Add("redirect_uri", s.config.JiraRedirectURI)
	params.Add("response_type", "code")
	params.Add("state", state)
	params.Add("scope", "read:jira-user read:jira-work offline_access")

	return fmt.Sprintf("%s/oauth/authorize?%s", s.config.JiraBaseURL, params.Encode())
}

// ValidateToken validates if a token is valid and not expired
func (s *AuthServiceImpl) ValidateToken(token *domain.OAuthToken) bool {
	return token.ExpiresAt.After(time.Now())
}

// exchangeCodeForToken performs the HTTP request to exchange code for access token
func (s *AuthServiceImpl) exchangeCodeForToken(ctx context.Context, code string) (*jiraTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", s.config.JiraRedirectURI)
	data.Set("client_id", s.config.JiraClientID)
	data.Set("client_secret", s.config.JiraClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", s.config.JiraBaseURL+"/oauth/token", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = data.Encode()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tokenResp jiraTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokenResp, nil
}

// refreshToken performs the HTTP request to refresh access token
func (s *AuthServiceImpl) refreshToken(ctx context.Context, refreshToken string) (*jiraTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", s.config.JiraClientID)
	data.Set("client_secret", s.config.JiraClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", s.config.JiraBaseURL+"/oauth/token", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = data.Encode()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tokenResp jiraTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokenResp, nil
}

// getJiraUserInfo retrieves user information from Jira API
func (s *AuthServiceImpl) getJiraUserInfo(ctx context.Context, accessToken string) (*jiraUserResponse, error) {
	userInfoURL := fmt.Sprintf("%s/me", s.config.JiraBaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userResp jiraUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &userResp, nil
}

// getOrCreateUser retrieves existing user or creates new one from Jira user info
func (s *AuthServiceImpl) getOrCreateUser(ctx context.Context, jiraUser *jiraUserResponse) (*domain.User, error) {
	// Try to get user by email first
	user, err := s.userRepo.GetByEmail(ctx, jiraUser.Email)
	if err == nil {
		return user, nil
	}

	// Create new user
	newUser := &domain.User{
		ID:        jiraUser.AccountID, // Use Jira account ID as user ID
		Name:      jiraUser.Name,
		Email:     jiraUser.Email,
		CreatedAt: time.Now(),
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return newUser, nil
}

// pauseUserAutomations pauses all active automations for a user
func (s *AuthServiceImpl) pauseUserAutomations(ctx context.Context, userID string) error {
	rules, err := s.automationRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user automations: %w", err)
	}

	for _, rule := range rules {
		if rule.Status == domain.AutomationStatusActive {
			rule.Status = domain.AutomationStatusPaused
			if err := s.automationRepo.Update(ctx, rule); err != nil {
				return fmt.Errorf("failed to pause automation %s: %w", rule.ID, err)
			}
		}
	}

	return nil
}

/*
// DISABLED: JWT generation
// generateJWT generates a JWT token for the given user ID
func (s *AuthServiceImpl) generateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(), // 24 hour expiry
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
*/