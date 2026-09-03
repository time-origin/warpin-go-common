package xiaohongshu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/time-origin/warpin-go-common/auth/oauth"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestAuthenticateWithSecret(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var body string
		switch request.URL.Path {
		case tokenPath:
			var payload map[string]string
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("decode token payload: %v", err)
			}
			if payload["app_secret"] != "secret" || payload["code"] != "auth-code" {
				t.Fatalf("unexpected token payload: %#v", payload)
			}
			body = `{"code":0,"success":true,"data":{"access_token":"access-token","open_id":"openid"}}`
		case userInfoPath:
			if request.Header.Get("Authorization") != "Bearer access-token" {
				t.Fatalf("unexpected authorization header")
			}
			body = `{"code":0,"success":true,"data":{"open_id":"openid","nickname":"昵称","avatar":"https://example.com/avatar.png"}}`
		default:
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		return jsonResponse(body), nil
	})}
	provider, err := New(Config{AppID: "app-id", AppSecret: "secret"}, client)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	identity, err := provider.Authenticate(context.Background(), oauth.Credential{Code: "auth-code"})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if identity.ProviderAppID != "app-id" || identity.Subject != "openid" || identity.Nickname != "昵称" {
		t.Fatalf("unexpected identity: %#v", identity)
	}
}

func TestAuthenticateWithPKCE(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host != "openaccount.beta.xiaohongshu.com" {
			t.Fatalf("unexpected beta host: %s", request.URL.Host)
		}
		if request.URL.Path == tokenPath {
			var payload map[string]string
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("decode token payload: %v", err)
			}
			if payload["code_verifier"] != "verifier" || payload["app_secret"] != "" {
				t.Fatalf("unexpected PKCE payload: %#v", payload)
			}
			return jsonResponse(`{"code":0,"success":true,"data":{"access_token":"access-token","open_id":"openid"}}`), nil
		}
		return jsonResponse(`{"code":0,"success":true,"data":{"open_id":"openid"}}`), nil
	})}
	provider, err := New(Config{AppID: "app-id", Beta: true}, client)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	if _, err := provider.Authenticate(context.Background(), oauth.Credential{Code: "auth-code", CodeVerifier: "verifier"}); err != nil {
		t.Fatalf("authenticate with PKCE: %v", err)
	}
}

func TestAuthenticateRequiresSecretOrVerifier(t *testing.T) {
	provider, err := New(Config{AppID: "app-id"}, &http.Client{})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	_, err = provider.Authenticate(context.Background(), oauth.Credential{Code: "auth-code"})
	if !errors.Is(err, oauth.ErrNotConfigured) {
		t.Fatalf("expected configuration error, got %v", err)
	}
}

func TestAuthenticateRejectsInvalidCode(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(`{"code":1003,"success":false,"msg":"authorization failed"}`), nil
	})}
	provider, err := New(Config{AppID: "app-id", AppSecret: "secret"}, client)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	_, err = provider.Authenticate(context.Background(), oauth.Credential{Code: "bad-code"})
	if !errors.Is(err, oauth.ErrInvalidCredential) {
		t.Fatalf("expected invalid credential, got %v", err)
	}
}

func TestAuthenticateRedactsTransportError(t *testing.T) {
	const (
		secret = "provider-secret-must-not-leak"
		code   = "authorization-code-must-not-leak"
	)
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("sensitive transport error: %s %s", secret, code)
	})}
	provider, err := New(Config{AppID: "app-id", AppSecret: secret}, client)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	_, err = provider.Authenticate(context.Background(), oauth.Credential{Code: code})
	if !errors.Is(err, oauth.ErrProviderUnavailable) {
		t.Fatalf("expected provider-unavailable error, got %v", err)
	}
	for _, sensitive := range []string{secret, code} {
		if strings.Contains(err.Error(), sensitive) {
			t.Fatalf("error leaked credential %q: %v", sensitive, err)
		}
	}
}

func TestAuthenticateRejectsOversizedResponse(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(strings.Repeat("x", maxResponseBytes+1)), nil
	})}
	provider, err := New(Config{AppID: "app-id", AppSecret: "secret"}, client)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	_, err = provider.Authenticate(context.Background(), oauth.Credential{Code: "auth-code"})
	if !errors.Is(err, oauth.ErrInvalidResponse) {
		t.Fatalf("expected invalid-response error, got %v", err)
	}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
