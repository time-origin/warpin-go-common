package oauth

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Registry is an immutable collection of identity providers.
type Registry struct {
	providers map[string]IdentityProvider
}

// NewRegistry validates providers and builds an immutable registry.
func NewRegistry(providers ...IdentityProvider) (*Registry, error) {
	registered := make(map[string]IdentityProvider, len(providers))
	for _, provider := range providers {
		if provider == nil {
			return nil, fmt.Errorf("%w: nil provider", ErrNotConfigured)
		}
		key := normalizeProviderKey(provider.Key())
		if key == "" {
			return nil, fmt.Errorf("%w: empty provider key", ErrNotConfigured)
		}
		if _, exists := registered[key]; exists {
			return nil, fmt.Errorf("%w: duplicate provider %q", ErrNotConfigured, key)
		}
		registered[key] = provider
	}
	return &Registry{providers: registered}, nil
}

// Authenticate delegates authentication to the named provider.
func (r *Registry) Authenticate(ctx context.Context, provider string, credential Credential) (Identity, error) {
	key := normalizeProviderKey(provider)
	implementation, ok := r.providers[key]
	if !ok {
		return Identity{}, fmt.Errorf("%w: %q", ErrProviderNotFound, key)
	}
	if strings.TrimSpace(credential.Code) == "" {
		return Identity{}, fmt.Errorf("%w: empty authorization code", ErrInvalidCredential)
	}
	identity, err := implementation.Authenticate(ctx, credential)
	if err != nil {
		return Identity{}, err
	}
	if strings.TrimSpace(identity.ProviderAppID) == "" || strings.TrimSpace(identity.Subject) == "" {
		return Identity{}, fmt.Errorf("%w: provider %q returned an incomplete identity", ErrInvalidResponse, key)
	}
	return identity, nil
}

// Providers returns registered provider keys in stable order.
func (r *Registry) Providers() []string {
	keys := make([]string, 0, len(r.providers))
	for key := range r.providers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func normalizeProviderKey(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}
