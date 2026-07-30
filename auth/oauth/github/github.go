package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	oauthcommon "github.com/time-origin/warpin-go-common/auth/oauth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// UserInfo holds the basic information for a GitHub user.
type UserInfo struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	AvatarURL string `json:"avatar_url"`
	Name      string `json:"name"`
	Email     string `json:"email"` // This will be populated from a separate API call.
}

// ExchangeCodeForToken exchanges an authorization code for a GitHub access token.
func ExchangeCodeForToken(ctx context.Context, code string, cfg oauthcommon.ProviderConfig) (*oauth2.Token, error) {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     github.Endpoint,
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	token, err := oauthConfig.Exchange(ctxTimeout, code)
	if err != nil {
		return nil, fmt.Errorf("github oauth exchange failed: %w", err)
	}

	return token, nil
}

// GetGitHubUserInfo fetches user information from the GitHub API using an access token.
func GetGitHubUserInfo(accessToken string) (*UserInfo, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is empty")
	}

	// 1. Get user's primary profile information
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to get user info: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info from github: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info, status: %s, body: %s", resp.Status, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response body: %w", err)
	}

	var userInfo UserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	// 2. Get user's email information
	email, err := getPrimaryVerifiedEmail(accessToken, client)
	if err != nil {
		// Log the error but don't fail the whole process if email is not found.
		// Some users may not have a public email.
		fmt.Printf("could not fetch github user email: %v\n", err)
	}
	userInfo.Email = email

	if userInfo.Email == "" {
		return nil, fmt.Errorf("could not find a primary, verified email for the user")
	}

	return &userInfo, nil
}

// getPrimaryVerifiedEmail fetches the user's emails and returns the primary, verified one.
func getPrimaryVerifiedEmail(accessToken string, client *http.Client) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request to get emails: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get emails from github: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get emails, status: %s, body: %s", resp.Status, string(bodyBytes))
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("failed to decode emails response: %w", err)
	}

	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	return "", fmt.Errorf("no primary verified email found")
}
