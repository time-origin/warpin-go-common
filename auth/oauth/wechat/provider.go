// Package wechat implements WeChat mobile-app authorization-code login.
package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/time-origin/warpin-go-common/auth/oauth"
)

const (
	ProviderKey       = "wechat"
	tokenEndpoint     = "https://api.weixin.qq.com/sns/oauth2/access_token"
	userInfoEndpoint  = "https://api.weixin.qq.com/sns/userinfo"
	defaultTimeout    = 10 * time.Second
	maxResponseBytes  = 1 << 20
	authorizationType = "authorization_code"
)

type Config struct {
	// AppID and AppSecret are supplied by the consuming application.
	// This package never loads configuration or credentials from the environment.
	AppID     string
	AppSecret string
	Timeout   time.Duration
}

type Provider struct {
	appID      string
	appSecret  string
	timeout    time.Duration
	httpClient *http.Client
}

var _ oauth.IdentityProvider = (*Provider)(nil)

func New(cfg Config, httpClient *http.Client) (*Provider, error) {
	appID := strings.TrimSpace(cfg.AppID)
	appSecret := strings.TrimSpace(cfg.AppSecret)
	if appID == "" {
		return nil, fmt.Errorf("%w: wechat app_id is empty", oauth.ErrNotConfigured)
	}
	if appSecret == "" {
		return nil, fmt.Errorf("%w: wechat app_secret is empty", oauth.ErrNotConfigured)
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	clientCopy := *httpClient
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Provider{
		appID:      appID,
		appSecret:  appSecret,
		timeout:    timeout,
		httpClient: &clientCopy,
	}, nil
}

func (p *Provider) Key() string { return ProviderKey }

func (p *Provider) Authenticate(ctx context.Context, credential oauth.Credential) (oauth.Identity, error) {
	code := strings.TrimSpace(credential.Code)
	if code == "" {
		return oauth.Identity{}, fmt.Errorf("%w: wechat authorization code is empty", oauth.ErrInvalidCredential)
	}

	requestCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	token, err := p.exchangeCode(requestCtx, code)
	if err != nil {
		return oauth.Identity{}, err
	}
	profile, err := p.getUserInfo(requestCtx, token.AccessToken, token.OpenID)
	if err != nil {
		return oauth.Identity{}, err
	}
	if profile.OpenID != token.OpenID {
		return oauth.Identity{}, fmt.Errorf("%w: wechat subject mismatch", oauth.ErrInvalidResponse)
	}
	unionID := profile.UnionID
	if unionID == "" {
		unionID = token.UnionID
	}
	return oauth.Identity{
		ProviderAppID: p.appID,
		Subject:       profile.OpenID,
		UnionID:       unionID,
		Nickname:      profile.Nickname,
		AvatarURL:     profile.AvatarURL,
	}, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	OpenID      string `json:"openid"`
	UnionID     string `json:"unionid"`
	ErrorCode   int    `json:"errcode"`
}

func (p *Provider) exchangeCode(ctx context.Context, code string) (tokenResponse, error) {
	query := url.Values{
		"appid":      {p.appID},
		"secret":     {p.appSecret},
		"code":       {code},
		"grant_type": {authorizationType},
	}
	var response tokenResponse
	if err := p.getJSON(ctx, tokenEndpoint+"?"+query.Encode(), &response); err != nil {
		return tokenResponse{}, err
	}
	if response.ErrorCode != 0 {
		return tokenResponse{}, wechatError("exchange code", response.ErrorCode)
	}
	if response.AccessToken == "" || response.OpenID == "" {
		return tokenResponse{}, fmt.Errorf("%w: wechat token response is incomplete", oauth.ErrInvalidResponse)
	}
	return response, nil
}

type userInfoResponse struct {
	OpenID    string `json:"openid"`
	UnionID   string `json:"unionid"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"headimgurl"`
	ErrorCode int    `json:"errcode"`
}

func (p *Provider) getUserInfo(ctx context.Context, accessToken, openID string) (userInfoResponse, error) {
	query := url.Values{
		"access_token": {accessToken},
		"openid":       {openID},
		"lang":         {"zh_CN"},
	}
	var response userInfoResponse
	if err := p.getJSON(ctx, userInfoEndpoint+"?"+query.Encode(), &response); err != nil {
		return userInfoResponse{}, err
	}
	if response.ErrorCode != 0 {
		return userInfoResponse{}, wechatError("get user info", response.ErrorCode)
	}
	if response.OpenID == "" {
		return userInfoResponse{}, fmt.Errorf("%w: wechat user response has no subject", oauth.ErrInvalidResponse)
	}
	return response, nil
}

func (p *Provider) getJSON(ctx context.Context, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("%w: create wechat request", oauth.ErrInvalidResponse)
	}
	response, err := p.httpClient.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("%w: wechat request canceled: %w", oauth.ErrProviderUnavailable, ctxErr)
		}
		return fmt.Errorf("%w: wechat request failed", oauth.ErrProviderUnavailable)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: wechat returned HTTP %d", oauth.ErrProviderUnavailable, response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("%w: read wechat response", oauth.ErrInvalidResponse)
	}
	if len(body) > maxResponseBytes {
		return fmt.Errorf("%w: wechat response is too large", oauth.ErrInvalidResponse)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("%w: decode wechat response", oauth.ErrInvalidResponse)
	}
	return nil
}

func wechatError(operation string, code int) error {
	category := oauth.ErrProviderUnavailable
	switch code {
	case 40029, 40014, 40163, 42001, 42003:
		category = oauth.ErrInvalidCredential
	case 40013, 40125, 48001:
		category = oauth.ErrNotConfigured
	}
	return fmt.Errorf("%w: wechat %s failed with code %d", category, operation, code)
}
