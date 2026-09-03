// Package xiaohongshu implements Xiaohongshu OAuth 2.0 login.
package xiaohongshu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/time-origin/warpin-go-common/auth/oauth"
)

const (
	ProviderKey       = "xiaohongshu"
	productionBaseURL = "https://openaccount.xiaohongshu.com"
	betaBaseURL       = "https://openaccount.beta.xiaohongshu.com"
	tokenPath         = "/api/sns/v1/oauth2/access_token"
	userInfoPath      = "/api/sns/v1/oauth2/batch_get_min_user_info"
	defaultTimeout    = 10 * time.Second
	maxResponseBytes  = 1 << 20
)

type Config struct {
	AppID     string        `mapstructure:"app_id"`
	AppSecret string        `mapstructure:"app_secret"`
	Beta      bool          `mapstructure:"beta"`
	Timeout   time.Duration `mapstructure:"timeout"`
}

type Provider struct {
	appID      string
	appSecret  string
	baseURL    string
	timeout    time.Duration
	httpClient *http.Client
}

var _ oauth.IdentityProvider = (*Provider)(nil)

// New creates a provider. AppSecret may be empty when the caller supplies a
// PKCE code verifier with each credential.
func New(cfg Config, httpClient *http.Client) (*Provider, error) {
	appID := strings.TrimSpace(cfg.AppID)
	if appID == "" {
		return nil, fmt.Errorf("%w: xiaohongshu app_id is empty", oauth.ErrNotConfigured)
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
	baseURL := productionBaseURL
	if cfg.Beta {
		baseURL = betaBaseURL
	}
	return &Provider{
		appID:      appID,
		appSecret:  strings.TrimSpace(cfg.AppSecret),
		baseURL:    baseURL,
		timeout:    timeout,
		httpClient: &clientCopy,
	}, nil
}

func (p *Provider) Key() string { return ProviderKey }

func (p *Provider) Authenticate(ctx context.Context, credential oauth.Credential) (oauth.Identity, error) {
	code := strings.TrimSpace(credential.Code)
	if code == "" {
		return oauth.Identity{}, fmt.Errorf("%w: xiaohongshu authorization code is empty", oauth.ErrInvalidCredential)
	}

	requestCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	token, err := p.exchangeCode(requestCtx, code, strings.TrimSpace(credential.CodeVerifier))
	if err != nil {
		return oauth.Identity{}, err
	}
	profile, err := p.getUserInfo(requestCtx, token.AccessToken)
	if err != nil {
		return oauth.Identity{}, err
	}
	if profile.OpenID != token.OpenID {
		return oauth.Identity{}, fmt.Errorf("%w: xiaohongshu subject mismatch", oauth.ErrInvalidResponse)
	}
	return oauth.Identity{
		ProviderAppID: p.appID,
		Subject:       profile.OpenID,
		Nickname:      profile.Nickname,
		AvatarURL:     profile.AvatarURL,
	}, nil
}

type tokenResponse struct {
	Code    int  `json:"code"`
	Success bool `json:"success"`
	Data    struct {
		AccessToken string `json:"access_token"`
		OpenID      string `json:"open_id"`
	} `json:"data"`
}

type tokenResponseData struct {
	AccessToken string
	OpenID      string
}

func (p *Provider) exchangeCode(ctx context.Context, code, codeVerifier string) (tokenResponseData, error) {
	payload := map[string]string{
		"app_id": p.appID,
		"code":   code,
	}
	if codeVerifier != "" {
		payload["code_verifier"] = codeVerifier
	} else if p.appSecret != "" {
		payload["app_secret"] = p.appSecret
	} else {
		return tokenResponseData{}, fmt.Errorf("%w: xiaohongshu requires app_secret or code_verifier", oauth.ErrNotConfigured)
	}

	var response tokenResponse
	if err := p.postJSON(ctx, p.baseURL+tokenPath, payload, "", &response); err != nil {
		return tokenResponseData{}, err
	}
	if !response.Success || response.Code != 0 {
		return tokenResponseData{}, xiaohongshuError("exchange code", response.Code)
	}
	if response.Data.AccessToken == "" || response.Data.OpenID == "" {
		return tokenResponseData{}, fmt.Errorf("%w: xiaohongshu token response is incomplete", oauth.ErrInvalidResponse)
	}
	return tokenResponseData{
		AccessToken: response.Data.AccessToken,
		OpenID:      response.Data.OpenID,
	}, nil
}

type userInfoResponse struct {
	Code    int  `json:"code"`
	Success bool `json:"success"`
	Data    struct {
		OpenID    string `json:"open_id"`
		Nickname  string `json:"nickname"`
		AvatarURL string `json:"avatar"`
	} `json:"data"`
}

type userInfoResponseData struct {
	OpenID    string
	Nickname  string
	AvatarURL string
}

func (p *Provider) getUserInfo(ctx context.Context, accessToken string) (userInfoResponseData, error) {
	var response userInfoResponse
	if err := p.postJSON(ctx, p.baseURL+userInfoPath, nil, accessToken, &response); err != nil {
		return userInfoResponseData{}, err
	}
	if !response.Success || response.Code != 0 {
		return userInfoResponseData{}, xiaohongshuError("get user info", response.Code)
	}
	if response.Data.OpenID == "" {
		return userInfoResponseData{}, fmt.Errorf("%w: xiaohongshu user response has no subject", oauth.ErrInvalidResponse)
	}
	return userInfoResponseData{
		OpenID:    response.Data.OpenID,
		Nickname:  response.Data.Nickname,
		AvatarURL: response.Data.AvatarURL,
	}, nil
}

func (p *Provider) postJSON(ctx context.Context, endpoint string, payload any, accessToken string, target any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("%w: encode xiaohongshu request", oauth.ErrInvalidResponse)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return fmt.Errorf("%w: create xiaohongshu request", oauth.ErrInvalidResponse)
	}
	request.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		request.Header.Set("Authorization", "Bearer "+accessToken)
	}
	response, err := p.httpClient.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("%w: xiaohongshu request canceled: %w", oauth.ErrProviderUnavailable, ctxErr)
		}
		return fmt.Errorf("%w: xiaohongshu request failed", oauth.ErrProviderUnavailable)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: xiaohongshu returned HTTP %d", oauth.ErrProviderUnavailable, response.StatusCode)
	}
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("%w: read xiaohongshu response", oauth.ErrInvalidResponse)
	}
	if len(responseBody) > maxResponseBytes {
		return fmt.Errorf("%w: xiaohongshu response is too large", oauth.ErrInvalidResponse)
	}
	if err := json.Unmarshal(responseBody, target); err != nil {
		return fmt.Errorf("%w: decode xiaohongshu response", oauth.ErrInvalidResponse)
	}
	return nil
}

func xiaohongshuError(operation string, code int) error {
	category := oauth.ErrProviderUnavailable
	switch code {
	case 1002, 1003:
		category = oauth.ErrInvalidCredential
	case 1001:
		category = oauth.ErrNotConfigured
	}
	return fmt.Errorf("%w: xiaohongshu %s failed with code %d", category, operation, code)
}
