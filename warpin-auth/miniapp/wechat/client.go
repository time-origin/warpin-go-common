// Package wechat wraps WeChat Mini Program APIs. Runtime credentials are injected
// by the consuming application; this package never reads environment or files.
package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/time-origin/warpin-go-common/warpin-auth/oauth"
)

const origin = "https://api.weixin.qq.com"

type Config struct {
	AppID     string
	AppSecret string
	Timeout   time.Duration
}

type Client struct {
	config  Config
	http    *http.Client
	mu      sync.Mutex
	token   string
	expires time.Time
}

type Session struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid,omitempty"`
	SessionKey string `json:"-"`
}

type PhoneNumber struct {
	PhoneNumber     string `json:"phoneNumber"`
	PurePhoneNumber string `json:"purePhoneNumber"`
	CountryCode     string `json:"countryCode"`
	Watermark       struct {
		AppID     string `json:"appid"`
		Timestamp int64  `json:"timestamp"`
	} `json:"watermark"`
}

// APIError exposes only the numeric upstream code, never errMsg or credentials.
type APIError struct{ Code int }

func (e *APIError) Error() string { return fmt.Sprintf("wechat miniapp API error %d", e.Code) }
func (e *APIError) Unwrap() error {
	switch e.Code {
	case 40029, 40163:
		return oauth.ErrInvalidCredential
	case 40013, 40125, 48001:
		return oauth.ErrNotConfigured
	default:
		return oauth.ErrProviderUnavailable
	}
}

func New(cfg Config, client *http.Client) (*Client, error) {
	cfg.AppID, cfg.AppSecret = strings.TrimSpace(cfg.AppID), strings.TrimSpace(cfg.AppSecret)
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return nil, oauth.ErrNotConfigured
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if client == nil {
		client = &http.Client{}
	}
	copy := *client
	copy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{config: cfg, http: &copy}, nil
}

func (c *Client) Code2Session(ctx context.Context, code string) (Session, error) {
	if strings.TrimSpace(code) == "" {
		return Session{}, oauth.ErrInvalidCredential
	}
	q := url.Values{"appid": {c.config.AppID}, "secret": {c.config.AppSecret}, "js_code": {code}, "grant_type": {"authorization_code"}}
	var out struct {
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		SessionKey string `json:"session_key"`
	}
	if err := c.request(ctx, http.MethodGet, "/sns/jscode2session?"+q.Encode(), nil, &out); err != nil {
		return Session{}, err
	}
	if out.OpenID == "" || out.SessionKey == "" {
		return Session{}, oauth.ErrInvalidResponse
	}
	return Session{OpenID: out.OpenID, UnionID: out.UnionID, SessionKey: out.SessionKey}, nil
}

// GetAccessToken caches per-client credentials. Construct one client per AppID.
func (c *Client) GetAccessToken(ctx context.Context, forceRefresh bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !forceRefresh && c.token != "" && time.Now().Before(c.expires) {
		return c.token, nil
	}
	var out struct {
		Token     string `json:"access_token"`
		ExpiresIn int    `json:"expires_in"`
	}
	err := c.request(ctx, http.MethodPost, "/cgi-bin/stable_token", map[string]any{"grant_type": "client_credential", "appid": c.config.AppID, "secret": c.config.AppSecret, "force_refresh": forceRefresh}, &out)
	if err != nil {
		return "", err
	}
	if out.Token == "" || out.ExpiresIn <= 0 || out.ExpiresIn > 86400 {
		return "", oauth.ErrInvalidResponse
	}
	c.token = out.Token
	// Keep a margin even for unusually short lifetimes returned by the provider.
	c.expires = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second * 9 / 10)
	return c.token, nil
}

func (c *Client) GetPhoneNumber(ctx context.Context, code string) (PhoneNumber, error) {
	if strings.TrimSpace(code) == "" {
		return PhoneNumber{}, oauth.ErrInvalidCredential
	}
	token, err := c.GetAccessToken(ctx, false)
	if err != nil {
		return PhoneNumber{}, err
	}
	for attempt := 0; attempt < 2; attempt++ {
		var out struct {
			Phone PhoneNumber `json:"phone_info"`
		}
		err = c.request(ctx, http.MethodPost, "/wxa/business/getuserphonenumber?"+url.Values{"access_token": {token}}.Encode(), map[string]string{"code": code}, &out)
		if err == nil {
			if out.Phone.PurePhoneNumber == "" || out.Phone.Watermark.AppID != c.config.AppID {
				return PhoneNumber{}, oauth.ErrInvalidResponse
			}
			return out.Phone, nil
		}
		var api *APIError
		if attempt != 0 || !errors.As(err, &api) || (api.Code != 40001 && api.Code != 40014 && api.Code != 42001) {
			return PhoneNumber{}, err
		}
		token, err = c.GetAccessToken(ctx, true)
		if err != nil {
			return PhoneNumber{}, err
		}
	}
	return PhoneNumber{}, err
}

func (c *Client) request(ctx context.Context, method, path string, body any, target any) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	var raw []byte
	var err error
	if body != nil {
		raw, err = json.Marshal(body)
		if err != nil {
			return oauth.ErrInvalidCredential
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, origin+path, bytes.NewReader(raw))
	if err != nil {
		return oauth.ErrInvalidCredential
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %w", oauth.ErrProviderUnavailable, ctx.Err())
		}
		return oauth.ErrProviderUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oauth.ErrProviderUnavailable
	}
	raw, err = io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return oauth.ErrInvalidResponse
	}
	var status struct {
		Code int `json:"errcode"`
	}
	if json.Unmarshal(raw, &status) != nil {
		return oauth.ErrInvalidResponse
	}
	if status.Code != 0 {
		return &APIError{Code: status.Code}
	}
	if json.Unmarshal(raw, target) != nil {
		return oauth.ErrInvalidResponse
	}
	return nil
}
