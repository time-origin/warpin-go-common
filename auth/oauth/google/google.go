package google

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	oauthcommon "github.com/time-origin/warpin-go-common/auth/oauth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// UserInfo represents the claims extracted from the ID Token.
type UserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// TokenResponse holds the tokens returned by the OAuth provider.
type TokenResponse struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
}

// ExchangeCodeForToken implements the OAuth 2.0 authorization code exchange step.
func ExchangeCodeForToken(ctx context.Context, code string, cfg oauthcommon.ProviderConfig) (*TokenResponse, error) {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       oauthcommon.GoogleDefaultScopes,
		Endpoint:     google.Endpoint,
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	token, err := oauthConfig.Exchange(ctxTimeout, code)
	if err != nil {
		return nil, fmt.Errorf("oauth exchange failed: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, errors.New("token response is missing id_token")
	}

	return &TokenResponse{
		AccessToken:  token.AccessToken,
		IDToken:      rawIDToken,
		RefreshToken: token.RefreshToken,
	}, nil
}

// ValidateAndParseIDToken validates the ID Token and parses it to extract user information.
func ValidateAndParseIDToken(
	ctx context.Context,
	provider *oidc.Provider,
	clientID string,
	rawIDToken string,
) (*UserInfo, error) {
	if rawIDToken == "" {
		return nil, errors.New("raw ID token is empty")
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})

	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	var userInfo UserInfo
	if err := idToken.Claims(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse ID token claims into UserInfo: %w", err)
	}

	if userInfo.Sub == "" {
		return nil, errors.New("ID token is missing user subject (sub)")
	}

	return &userInfo, nil
}
