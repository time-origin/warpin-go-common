package douyin

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
		if err := request.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		var body string
		switch request.URL.Path {
		case "/oauth/access_token/":
			if request.Form.Get("code") != "auth-code" || request.Form.Get("client_secret") != "secret" {
				t.Fatalf("unexpected token form: %#v", request.Form)
			}
			body = `{"data":{"access_token":"access-token","open_id":"openid","error_code":0},"message":"success"}`
		case "/oauth/userinfo/":
			body = `{"err_no":0,"data":{"open_id":"openid","union_id":"unionid","nickname":"昵称","avatar":"https://example.com/avatar.png","client_key":"client-key"}}`
		default:
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		return jsonResponse(body), nil
	})}
	provider, err := New(Config{ClientKey: "client-key", ClientSecret: "secret"}, client)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	identity, err := provider.Authenticate(context.Background(), oauth.Credential{Code: "auth-code"})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if identity.ProviderAppID != "client-key" || identity.Subject != "openid" || identity.UnionID != "unionid" || identity.Nickname != "昵称" {
		t.Fatalf("unexpected identity: %#v", identity)
	}
}

func TestAuthenticateRejectsExpiredCode(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(`{"data":{"error_code":10007,"description":"authorization code expired"}}`), nil
	})}
	provider, err := New(Config{ClientKey: "client-key", ClientSecret: "secret"}, client)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	_, err = provider.Authenticate(context.Background(), oauth.Credential{Code: "expired-code"})
	if !errors.Is(err, oauth.ErrInvalidCredential) {
		t.Fatalf("expected invalid credential, got %v", err)
	}
}

func TestAuthenticateRejectsSubjectMismatch(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path == "/oauth/access_token/" {
			return jsonResponse(`{"data":{"access_token":"access-token","open_id":"openid","error_code":0}}`), nil
		}
		return jsonResponse(`{"err_no":0,"data":{"open_id":"another-user","client_key":"client-key"}}`), nil
	})}
	provider, err := New(Config{ClientKey: "client-key", ClientSecret: "secret"}, client)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	_, err = provider.Authenticate(context.Background(), oauth.Credential{Code: "auth-code"})
	if !errors.Is(err, oauth.ErrInvalidResponse) {
		t.Fatalf("expected invalid response, got %v", err)
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
