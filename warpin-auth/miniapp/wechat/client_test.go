package wechat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/time-origin/warpin-go-common/warpin-auth/oauth"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func reply(body string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func TestLoginPhoneAndConcurrentTokenCache(t *testing.T) {
	var mu sync.Mutex
	tokens := 0
	c, err := New(Config{AppID: "test-app", AppSecret: "never-log-this"}, &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.weixin.qq.com" {
			t.Error("unexpected host")
		}
		switch r.URL.Path {
		case "/sns/jscode2session":
			if r.URL.Query().Get("js_code") != "login-code" || r.URL.Query().Get("secret") != "never-log-this" {
				t.Error("invalid session query")
			}
			return reply(`{"openid":"openid-1","session_key":"private-session","unionid":"union-1"}`), nil
		case "/cgi-bin/stable_token":
			mu.Lock()
			tokens++
			mu.Unlock()
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if r.Method != "POST" || body["appid"] != "test-app" || body["secret"] != "never-log-this" {
				t.Error("invalid token request")
			}
			return reply(`{"access_token":"access","expires_in":7200}`), nil
		case "/wxa/business/getuserphonenumber":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if r.Method != "POST" || body["code"] != "phone-code" || r.URL.Query().Get("access_token") != "access" {
				t.Error("invalid phone request")
			}
			return reply(`{"errcode":0,"phone_info":{"purePhoneNumber":"13800138000","countryCode":"86","watermark":{"appid":"test-app"}}}`), nil
		}
		return nil, errors.New("unexpected path")
	})})
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Code2Session(context.Background(), "login-code")
	if err != nil || s.OpenID != "openid-1" || s.SessionKey != "private-session" {
		t.Fatalf("session failed: %v", err)
	}
	raw, _ := json.Marshal(s)
	if strings.Contains(string(raw), "private-session") {
		t.Fatal("session key serialized")
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Go(func() {
			p, err := c.GetPhoneNumber(context.Background(), "phone-code")
			if err != nil || p.PurePhoneNumber != "13800138000" {
				t.Errorf("phone failed: %v", err)
			}
		})
	}
	wg.Wait()
	if tokens != 1 {
		t.Fatalf("token requested %d times", tokens)
	}
}

func TestProviderFailuresAndRefresh(t *testing.T) {
	for _, body := range []string{`{"errcode":40029,"errmsg":"secret-code"}`, `{}`, `[]`, `{"openid":"id"}`} {
		c, _ := New(Config{AppID: "app", AppSecret: "secret"}, &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { return reply(body), nil })})
		_, err := c.Code2Session(context.Background(), "code")
		if err == nil || strings.Contains(err.Error(), "secret-code") {
			t.Fatal("invalid response accepted or provider message leaked")
		}
	}
	tokens, phones := 0, 0
	c, _ := New(Config{AppID: "app", AppSecret: "secret"}, &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/cgi-bin/stable_token" {
			tokens++
			return reply(`{"access_token":"access","expires_in":7200}`), nil
		}
		phones++
		if phones == 1 {
			return reply(`{"errcode":42001}`), nil
		}
		return reply(`{"phone_info":{"purePhoneNumber":"13800138000","watermark":{"appid":"wrong-app"}}}`), nil
	})})
	if _, err := c.GetPhoneNumber(context.Background(), "phone"); !errors.Is(err, oauth.ErrInvalidResponse) {
		t.Fatal("wrong watermark accepted")
	}
	if tokens != 2 || phones != 2 {
		t.Fatal("token refresh not retried exactly once")
	}
	c, _ = New(Config{AppID: "app", AppSecret: "secret"}, &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { return nil, errors.New("https://secret-url") })})
	_, err := c.Code2Session(context.Background(), "code")
	if err == nil || strings.Contains(err.Error(), "secret-url") {
		t.Fatal("transport detail leaked")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c, _ = New(Config{AppID: "app", AppSecret: "secret"}, nil)
	_, err = c.Code2Session(ctx, "code")
	if !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost")
	}
}
