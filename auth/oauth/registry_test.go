package oauth

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type stubProvider struct {
	key      string
	identity Identity
}

func (p stubProvider) Key() string { return p.key }

func (p stubProvider) Authenticate(context.Context, Credential) (Identity, error) {
	return p.identity, nil
}

func TestRegistryAuthenticatesAndListsProviders(t *testing.T) {
	registry, err := NewRegistry(
		stubProvider{key: "wechat", identity: Identity{ProviderAppID: "wx-app", Subject: "wx-user"}},
		stubProvider{key: "douyin", identity: Identity{ProviderAppID: "dy-app", Subject: "dy-user"}},
	)
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}

	identity, err := registry.Authenticate(context.Background(), " WeChat ", Credential{Code: "code"})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if identity.Subject != "wx-user" {
		t.Fatalf("unexpected identity: %#v", identity)
	}
	if actual := registry.Providers(); !reflect.DeepEqual(actual, []string{"douyin", "wechat"}) {
		t.Fatalf("unexpected provider list: %#v", actual)
	}
}

func TestRegistryRejectsDuplicateProvider(t *testing.T) {
	_, err := NewRegistry(stubProvider{key: "wechat"}, stubProvider{key: "WECHAT"})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected configuration error, got %v", err)
	}
}

func TestRegistryRejectsUnknownProviderAndEmptyCode(t *testing.T) {
	registry, err := NewRegistry(stubProvider{key: "wechat"})
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}

	if _, err := registry.Authenticate(context.Background(), "douyin", Credential{Code: "code"}); !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("expected provider-not-found error, got %v", err)
	}
	if _, err := registry.Authenticate(context.Background(), "wechat", Credential{}); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("expected invalid-credential error, got %v", err)
	}
}

func TestRegistryRejectsIncompleteIdentity(t *testing.T) {
	registry, err := NewRegistry(stubProvider{key: "wechat"})
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}

	if _, err := registry.Authenticate(context.Background(), "wechat", Credential{Code: "code"}); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("expected invalid-response error, got %v", err)
	}
}
