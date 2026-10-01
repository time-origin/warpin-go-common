// Package oauth defines reusable third-party identity authentication contracts.
// It does not own application users, account binding, sessions, or business JWTs.
package oauth

import (
	"context"
	"errors"
)

var (
	ErrInvalidCredential   = errors.New("oauth credential is invalid")
	ErrProviderUnavailable = errors.New("oauth provider is unavailable")
	ErrInvalidResponse     = errors.New("oauth provider returned an invalid response")
	ErrNotConfigured       = errors.New("oauth provider is not configured")
	ErrProviderNotFound    = errors.New("oauth provider is not registered")
)

// Credential contains the short-lived proof returned by a provider's client SDK.
type Credential struct {
	Code         string
	CodeVerifier string
	Nonce        string
}

// Identity is the provider-neutral identity returned after authentication.
// Subject is the stable identifier within ProviderAppID. UnionID is optional.
type Identity struct {
	ProviderAppID string
	Subject       string
	UnionID       string
	Nickname      string
	AvatarURL     string
	Email         string
	EmailVerified bool
}

// IdentityProvider validates a provider credential and returns its identity.
// Implementations must not expose provider access or refresh tokens.
type IdentityProvider interface {
	Key() string
	Authenticate(ctx context.Context, credential Credential) (Identity, error)
}
