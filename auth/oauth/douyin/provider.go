// Package douyin implements Douyin authorization-code login.
package douyin

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
	ProviderKey      = "douyin"
	tokenEndpoint    = "https://open.douyin.com/oauth/access_token/"
	userInfoEndpoint = "https://open.douyin.com/oauth/userinfo/"
	defaultTimeout   = 10 * time.Second
	maxResponseBytes = 1 << 20
)

type Config struct {
	// ClientKey and ClientSecret are supplied by the consuming application.
	// This package never loads configuration or credentials from the environment.
	ClientKey    string
	ClientSecret string
	Timeout      time.Duration
}

type Provider struct {
	clientKey    string
	clientSecret string
	timeout      time.Duration
	httpClient   *http.Client
}

var _ oauth.IdentityProvider = (*Provider)(nil)

func New(cfg Config, httpClient *http.Client) (*Provider, error) {
	clientKey := strings.TrimSpace(cfg.ClientKey)
	clientSecret := strings.TrimSpace(cfg.ClientSecret)
	if clientKey == "" {
		return nil, fmt.Errorf("%w: douyin client_key is empty", oauth.ErrNotConfigured)
	}
	if clientSecret == "" {
		return nil, fmt.Errorf("%w: douyin client_secret is empty", oauth.ErrNotConfigured)
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
		clientKey:    clientKey,
		clientSecret: clientSecret,
		timeout:      timeout,
		httpClient:   &clientCopy,
	}, nil
}

func (p *Provider) Key() string { return ProviderKey }

func (p *Provider) Authenticate(ctx context.Context, credential oauth.Credential) (oauth.Identity, error) {
	code := strings.TrimSpace(credential.Code)
	if code == "" {
		return oauth.Identity{}, fmt.Errorf("%w: douyin authorization code is empty", oauth.ErrInvalidCredential)
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
		return oauth.Identity{}, fmt.Errorf("%w: douyin subject mismatch", oauth.ErrInvalidResponse)
	}
	if profile.ClientKey != "" && profile.ClientKey != p.clientKey {
		return oauth.Identity{}, fmt.Errorf("%w: douyin client key mismatch", oauth.ErrInvalidResponse)
	}
	return oauth.Identity{
		ProviderAppID: p.clientKey,
		Subject:       profile.OpenID,
		UnionID:       profile.UnionID,
		Nickname:      profile.Nickname,
		AvatarURL:     profile.AvatarURL,
	}, nil
}

type tokenResponse struct {
	Data struct {
		AccessToken string `json:"access_token"`
		OpenID      string `json:"open_id"`
		ErrorCode   int    `json:"error_code"`
	} `json:"data"`
}

func (p *Provider) exchangeCode(ctx context.Context, code string) (tokenResponseData, error) {
	form := url.Values{
		"client_key":    {p.clientKey},
		"client_secret": {p.clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	}
	var response tokenResponse
	if err := p.postForm(ctx, tokenEndpoint, form, &response); err != nil {
		return tokenResponseData{}, err
	}
	if response.Data.ErrorCode != 0 {
		return tokenResponseData{}, douyinTokenError(response.Data.ErrorCode)
	}
	if response.Data.AccessToken == "" || response.Data.OpenID == "" {
		return tokenResponseData{}, fmt.Errorf("%w: douyin token response is incomplete", oauth.ErrInvalidResponse)
	}
	return tokenResponseData{
		AccessToken: response.Data.AccessToken,
		OpenID:      response.Data.OpenID,
	}, nil
}

type tokenResponseData struct {
	AccessToken string
	OpenID      string
}

type userInfoResponse struct {
	ErrorNumber int `json:"err_no"`
	Data        struct {
		OpenID    string `json:"open_id"`
		UnionID   string `json:"union_id"`
		Nickname  string `json:"nickname"`
		AvatarURL string `json:"avatar"`
		ClientKey string `json:"client_key"`
	} `json:"data"`
}

func (p *Provider) getUserInfo(ctx context.Context, accessToken, openID string) (userInfoResponseData, error) {
	form := url.Values{
		"access_token": {accessToken},
		"open_id":      {openID},
	}
	var response userInfoResponse
	if err := p.postForm(ctx, userInfoEndpoint, form, &response); err != nil {
		return userInfoResponseData{}, err
	}
	if response.ErrorNumber != 0 {
		return userInfoResponseData{}, douyinUserInfoError(response.ErrorNumber)
	}
	if response.Data.OpenID == "" {
		return userInfoResponseData{}, fmt.Errorf("%w: douyin user response has no subject", oauth.ErrInvalidResponse)
	}
	return userInfoResponseData{
		OpenID:    response.Data.OpenID,
		UnionID:   response.Data.UnionID,
		Nickname:  response.Data.Nickname,
		AvatarURL: response.Data.AvatarURL,
		ClientKey: response.Data.ClientKey,
	}, nil
}

type userInfoResponseData struct {
	OpenID    string
	UnionID   string
	Nickname  string
	AvatarURL string
	ClientKey string
}

func (p *Provider) postForm(ctx context.Context, endpoint string, form url.Values, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("%w: create douyin request", oauth.ErrInvalidResponse)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := p.httpClient.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("%w: douyin request canceled: %w", oauth.ErrProviderUnavailable, ctxErr)
		}
		return fmt.Errorf("%w: douyin request failed", oauth.ErrProviderUnavailable)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: douyin returned HTTP %d", oauth.ErrProviderUnavailable, response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("%w: read douyin response", oauth.ErrInvalidResponse)
	}
	if len(body) > maxResponseBytes {
		return fmt.Errorf("%w: douyin response is too large", oauth.ErrInvalidResponse)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("%w: decode douyin response", oauth.ErrInvalidResponse)
	}
	return nil
}

func douyinTokenError(code int) error {
	category := oauth.ErrProviderUnavailable
	switch code {
	case 10002, 10007:
		category = oauth.ErrInvalidCredential
	case 10003, 10013, 10014:
		category = oauth.ErrNotConfigured
	}
	return fmt.Errorf("%w: douyin exchange code failed with code %d", category, code)
}

func douyinUserInfoError(code int) error {
	category := oauth.ErrProviderUnavailable
	switch code {
	case 28001003, 28001007, 28001008:
		category = oauth.ErrInvalidCredential
	case 28001014, 28001016, 28001018, 28001019:
		category = oauth.ErrNotConfigured
	}
	return fmt.Errorf("%w: douyin get user info failed with code %d", category, code)
}
