package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CLAOJ/claoj-go/config"
	"github.com/golang-jwt/jwt/v5"
)

// OAuthConfig holds provider configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// Provider type
type Provider string

const (
	ProviderGoogle Provider = "google"
	ProviderGitHub Provider = "github"
)

// Token represents an OAuth token
type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

// UserInfo represents authenticated user info
type UserInfo struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Provider  string `json:"provider"`
}

// GetAuthURL returns the OAuth authorization URL for the provider
func GetAuthURL(provider Provider, state string) (string, error) {
	cfg := getProviderConfig(provider)
	if cfg == nil {
		return "", errors.New("provider not configured")
	}

	switch provider {
	case ProviderGoogle:
		return googleAuthURL(cfg, state), nil
	case ProviderGitHub:
		return githubAuthURL(cfg, state), nil
	default:
		return "", errors.New("unsupported provider")
	}
}

func googleAuthURL(cfg *OAuthConfig, state string) string {
	params := url.Values{}
	params.Set("client_id", cfg.ClientID)
	params.Set("redirect_uri", cfg.RedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(cfg.Scopes, " "))
	params.Set("state", state)
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")

	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

func githubAuthURL(cfg *OAuthConfig, state string) string {
	params := url.Values{}
	params.Set("client_id", cfg.ClientID)
	params.Set("redirect_uri", cfg.RedirectURL)
	params.Set("scope", strings.Join(cfg.Scopes, " "))
	params.Set("state", state)

	return "https://github.com/login/oauth/authorize?" + params.Encode()
}

// ExchangeCode exchanges the authorization code for a token
func ExchangeCode(ctx context.Context, provider Provider, code string) (*Token, error) {
	cfg := getProviderConfig(provider)
	if cfg == nil {
		return nil, errors.New("provider not configured")
	}

	switch provider {
	case ProviderGoogle:
		return googleExchange(ctx, cfg, code)
	case ProviderGitHub:
		return githubExchange(ctx, cfg, code)
	default:
		return nil, errors.New("unsupported provider")
	}
}

func googleExchange(ctx context.Context, cfg *OAuthConfig, code string) (*Token, error) {
	data := url.Values{}
	data.Set("client_id", cfg.ClientID)
	data.Set("client_secret", cfg.ClientSecret)
	data.Set("redirect_uri", cfg.RedirectURL)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}

	token.Expiry = time.Now().Add(time.Duration(token.Expiry.Second()) * time.Second)
	return &token, nil
}

func githubExchange(ctx context.Context, cfg *OAuthConfig, code string) (*Token, error) {
	data := url.Values{}
	data.Set("client_id", cfg.ClientID)
	data.Set("client_secret", cfg.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", cfg.RedirectURL)

	resp, err := http.PostForm("https://github.com/login/oauth/access_token", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// GitHub returns URL-encoded response
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}

	token := &Token{
		AccessToken: values.Get("access_token"),
		TokenType:   values.Get("token_type"),
	}

	return token, nil
}

// GetUserInfo fetches user info from the provider
func GetUserInfo(ctx context.Context, provider Provider, token *Token) (*UserInfo, error) {
	switch provider {
	case ProviderGoogle:
		return googleUserInfo(token.AccessToken)
	case ProviderGitHub:
		return githubUserInfo(token.AccessToken)
	default:
		return nil, errors.New("unsupported provider")
	}
}

func googleUserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var googleUser struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := json.Unmarshal(body, &googleUser); err != nil {
		return nil, err
	}

	return &UserInfo{
		ID:        googleUser.Sub,
		Email:     googleUser.Email,
		Name:      googleUser.Name,
		AvatarURL: googleUser.Picture,
		Provider:  "google",
	}, nil
}

func githubUserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var githubUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &githubUser); err != nil {
		return nil, err
	}

	// If email is not public, fetch it separately
	email := githubUser.Email
	if email == "" {
		email, _ = getPrimaryGitHubEmail(accessToken)
	}

	return &UserInfo{
		ID:        fmt.Sprintf("%d", githubUser.ID),
		Email:     email,
		Name:      githubUser.Name,
		AvatarURL: githubUser.AvatarURL,
		Provider:  "github",
	}, nil
}

func getPrimaryGitHubEmail(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	return "", errors.New("no verified primary email found")
}

func getProviderConfig(provider Provider) *OAuthConfig {
	switch provider {
	case ProviderGoogle:
		cfg := config.C.OAuth.Google
		if !cfg.Enabled || cfg.ClientID == "" {
			return nil
		}
		return &OAuthConfig{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
		}
	case ProviderGitHub:
		cfg := config.C.OAuth.GitHub
		if !cfg.Enabled || cfg.ClientID == "" {
			return nil
		}
		return &OAuthConfig{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
		}
	default:
		return nil
	}
}

// GenerateStateToken creates a JWT state token for CSRF protection
func GenerateStateToken(secret string) (string, error) {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(10 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// VerifyStateToken verifies the state token
func VerifyStateToken(tokenString, secret string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return err
	}
	if !token.Valid {
		return errors.New("invalid state token")
	}
	return nil
}
