package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
	"go.uber.org/zap"
)

// OAuth2Config holds the client credentials for supported OAuth2 providers.
type OAuth2Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	RedirectBaseURL    string
}

// OAuth2Service handles OAuth2 authentication flows for Google and GitHub.
type OAuth2Service struct {
	config   OAuth2Config
	userRepo ports.UserRepository
	jwtSvc   *JWTService
	log      *zap.Logger
}

// googleUserInfo represents the response from Google's userinfo endpoint.
type googleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	VerifiedEmail bool   `json:"verified_email"`
}

// githubUserInfo represents the response from GitHub's user endpoint.
type githubUserInfo struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// googleTokenResponse represents the response from Google's token endpoint.
type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// githubTokenResponse represents the response from GitHub's token endpoint.
type githubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// NewOAuth2Service creates a new OAuth2Service instance.
func NewOAuth2Service(cfg OAuth2Config, userRepo ports.UserRepository, jwtSvc *JWTService, log *zap.Logger) *OAuth2Service {
	log.Info("OAuth2 service initialized",
		zap.String("redirect_base_url", cfg.RedirectBaseURL),
	)

	return &OAuth2Service{
		config:   cfg,
		userRepo: userRepo,
		jwtSvc:   jwtSvc,
		log:      log,
	}
}

// GetAuthorizationURL returns the OAuth2 authorization URL for the specified provider.
// Supported providers: "google", "github".
func (s *OAuth2Service) GetAuthorizationURL(provider string) (string, error) {
	switch provider {
	case "google":
		params := url.Values{
			"client_id":     {s.config.GoogleClientID},
			"redirect_uri":  {s.config.RedirectBaseURL + "/auth/callback/google"},
			"response_type": {"code"},
			"scope":         {"openid email profile"},
			"access_type":   {"offline"},
		}
		authURL := "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
		s.log.Debug("generated Google authorization URL")
		return authURL, nil

	case "github":
		params := url.Values{
			"client_id":    {s.config.GitHubClientID},
			"redirect_uri": {s.config.RedirectBaseURL + "/auth/callback/github"},
			"scope":        {"user:email"},
		}
		authURL := "https://github.com/login/oauth/authorize?" + params.Encode()
		s.log.Debug("generated GitHub authorization URL")
		return authURL, nil

	default:
		return "", fmt.Errorf("unsupported OAuth2 provider: %s", provider)
	}
}

// HandleCallback exchanges the authorization code for a token, fetches user info
// from the provider, finds or creates the user in the database, and returns
// the user along with access and refresh JWT tokens.
func (s *OAuth2Service) HandleCallback(ctx context.Context, provider, code string) (*domain.User, string, string, error) {
	var email, name string

	switch provider {
	case "google":
		e, n, err := s.handleGoogleCallback(ctx, code)
		if err != nil {
			return nil, "", "", fmt.Errorf("google callback failed: %w", err)
		}
		email, name = e, n

	case "github":
		e, n, err := s.handleGitHubCallback(ctx, code)
		if err != nil {
			return nil, "", "", fmt.Errorf("github callback failed: %w", err)
		}
		email, name = e, n

	default:
		return nil, "", "", fmt.Errorf("unsupported OAuth2 provider: %s", provider)
	}

	// Find or create user
	user, err := s.findOrCreateUser(ctx, email, name)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to find or create user: %w", err)
	}

	// Generate tokens
	accessToken, err := s.jwtSvc.GenerateAccessToken(user)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtSvc.GenerateRefreshToken(user)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	s.log.Info("OAuth2 login successful",
		zap.String("provider", provider),
		zap.String("user_id", user.ID),
		zap.String("email", user.Email),
	)

	return user, accessToken, refreshToken, nil
}

// handleGoogleCallback exchanges the code with Google and fetches user info.
func (s *OAuth2Service) handleGoogleCallback(ctx context.Context, code string) (string, string, error) {
	// Exchange authorization code for access token
	data := url.Values{
		"code":          {code},
		"client_id":     {s.config.GoogleClientID},
		"client_secret": {s.config.GoogleClientSecret},
		"redirect_uri":  {s.config.RedirectBaseURL + "/auth/callback/google"},
		"grant_type":    {"authorization_code"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("google token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp googleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", "", fmt.Errorf("failed to decode token response: %w", err)
	}

	// Fetch user info
	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create userinfo request: %w", err)
	}
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(userResp.Body)
		return "", "", fmt.Errorf("google userinfo failed (status %d): %s", userResp.StatusCode, string(body))
	}

	var userInfo googleUserInfo
	if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
		return "", "", fmt.Errorf("failed to decode user info: %w", err)
	}

	if userInfo.Email == "" {
		return "", "", fmt.Errorf("google user has no email")
	}

	return userInfo.Email, userInfo.Name, nil
}

// handleGitHubCallback exchanges the code with GitHub and fetches user info.
func (s *OAuth2Service) handleGitHubCallback(ctx context.Context, code string) (string, string, error) {
	// Exchange authorization code for access token
	data := url.Values{
		"code":          {code},
		"client_id":     {s.config.GitHubClientID},
		"client_secret": {s.config.GitHubClientSecret},
		"redirect_uri":  {s.config.RedirectBaseURL + "/auth/callback/github"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("github token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp githubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", "", fmt.Errorf("github returned empty access token")
	}

	// Fetch user info
	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create user request: %w", err)
	}
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	userReq.Header.Set("Accept", "application/vnd.github.v3+json")

	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(userResp.Body)
		return "", "", fmt.Errorf("github user info failed (status %d): %s", userResp.StatusCode, string(body))
	}

	var userInfo githubUserInfo
	if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
		return "", "", fmt.Errorf("failed to decode user info: %w", err)
	}

	email := userInfo.Email
	if email == "" {
		// GitHub users can have private emails; use login as fallback
		email = userInfo.Login + "@github.com"
	}

	name := userInfo.Name
	if name == "" {
		name = userInfo.Login
	}

	return email, name, nil
}

// findOrCreateUser looks up a user by email. If not found, creates a new user
// with the "user" role and "Active" status.
func (s *OAuth2Service) findOrCreateUser(ctx context.Context, email, name string) (*domain.User, error) {
	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil && existingUser != nil {
		s.log.Debug("existing user found for OAuth2 login",
			zap.String("user_id", existingUser.ID),
			zap.String("email", email),
		)
		return existingUser, nil
	}

	// Create a new user
	newUser := &domain.User{
		ID:        uuid.New().String(),
		Name:      name,
		Email:     email,
		Password:  "", // OAuth2 users have no password
		Role:      domain.UserRoleUser,
		Status:    "Active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.userRepo.Save(ctx, newUser); err != nil {
		s.log.Error("failed to create OAuth2 user",
			zap.String("email", email),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.log.Info("new OAuth2 user created",
		zap.String("user_id", newUser.ID),
		zap.String("email", email),
	)

	return newUser, nil
}
