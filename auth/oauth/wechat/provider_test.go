package wechat

import (
	"context"
	"errors"
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

func TestAuthenticate(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var body string
		switch request.URL.Path {
		case "/sns/oauth2/access_token":
			if request.URL.Query().Get("code") != "auth-code" || request.URL.Query().Get("secret") != "secret" {
				t.Fatalf("unexpected token request: %s", request.URL.RawQuery)
			}
			body = `{"access_token":"access-token","openid":"openid","unionid":"unionid"}`
		case "/sns/userinfo":
			body = `{"openid":"openid","unionid":"unionid","nickname":"昵称","headimgurl":"https://example.com/avatar.png"}`
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
	if identity.ProviderAppID != "app-id" || identity.Subject != "openid" || identity.UnionID != "unionid" || identity.Nickname != "昵称" {
		t.Fatalf("unexpected identity: %#v", identity)
	}
}

func TestAuthenticateRejectsInvalidCode(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(`{"errcode":40029,"errmsg":"invalid code"}`), nil
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

func TestNewRequiresConfiguration(t *testing.T) {
	if _, err := New(Config{}, nil); !errors.Is(err, oauth.ErrNotConfigured) {
		t.Fatalf("expected configuration error, got %v", err)
	}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
