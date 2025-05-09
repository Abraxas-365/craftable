package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Abraxas-365/craftale/auth"
	"github.com/Abraxas-365/craftale/errx"
)

const (
	// Google OAuth endpoints
	googleAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
)

// Define Google-specific error codes
var (
	providerErrors = errx.NewRegistry("GOOGLE_PROVIDER")

	ErrInvalidGrant    = providerErrors.Register("INVALID_GRANT", errx.TypeBadRequest, 400, "Invalid grant")
	ErrInvalidToken    = providerErrors.Register("INVALID_TOKEN", errx.TypeBadRequest, 400, "Invalid token")
	ErrAPIRequest      = providerErrors.Register("API_REQUEST_FAILED", errx.TypeExternal, 500, "Google API request failed")
	ErrResponseParsing = providerErrors.Register("RESPONSE_PARSING_FAILED", errx.TypeInternal, 500, "Failed to parse Google API response")
)

// GoogleUserInfo extends BasicAuthUserInfo with Google-specific fields
type GoogleUserInfo struct {
	auth.BasicAuthUserInfo
	VerifiedEmail bool   `json:"verified_email"`
	Locale        string `json:"locale"`
	FamilyName    string `json:"family_name"`
	GivenName     string `json:"given_name"`
	Hd            string `json:"hd"` // G Suite domain
}

// GoogleProvider implements the OAuthProvider interface for Google
type GoogleProvider struct {
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
	httpClient   *http.Client
}

// NewGoogleProvider creates a new Google OAuth provider
func NewGoogleProvider(clientID, clientSecret, redirectURI string) *GoogleProvider {
	return &GoogleProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		scopes:       []string{"openid", "profile", "email"},
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// WithScopes adds additional scopes to the provider
func (p *GoogleProvider) WithScopes(scopes ...string) *GoogleProvider {
	p.scopes = append(p.scopes, scopes...)
	return p
}

// WithHTTPClient sets a custom HTTP client
func (p *GoogleProvider) WithHTTPClient(client *http.Client) *GoogleProvider {
	p.httpClient = client
	return p
}

// GetAuthURL returns the Google authorization URL
func (p *GoogleProvider) GetAuthURL(state string) string {
	params := url.Values{}
	params.Add("client_id", p.clientID)
	params.Add("redirect_uri", p.redirectURI)
	params.Add("response_type", "code")
	params.Add("scope", strings.Join(p.scopes, " "))
	params.Add("access_type", "offline")
	params.Add("prompt", "consent") // Force consent to get refresh token
	params.Add("state", state)

	return fmt.Sprintf("%s?%s", googleAuthURL, params.Encode())
}

// ExchangeCode exchanges an authorization code for tokens
func (p *GoogleProvider) ExchangeCode(ctx context.Context, code string) (*auth.OAuthToken, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("redirect_uri", p.redirectURI)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, "POST", googleTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, providerErrors.New(ErrAPIRequest).WithCause(err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, providerErrors.New(ErrAPIRequest).
			WithDetail("error", err.Error()).
			WithCause(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, providerErrors.New(ErrAPIRequest).
			WithDetail("error", "failed to read response body").
			WithCause(err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, providerErrors.New(ErrAPIRequest).
				WithDetail("status_code", resp.StatusCode).
				WithDetail("body", string(body))
		}

		// Handle specific error types
		if errorResp.Error == "invalid_grant" {
			return nil, providerErrors.New(ErrInvalidGrant).
				WithDetail("error", errorResp.Description)
		}

		return nil, providerErrors.New(ErrAPIRequest).
			WithDetail("error", errorResp.Error).
			WithDetail("description", errorResp.Description)
	}

	// Parse token response
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, providerErrors.New(ErrResponseParsing).
			WithDetail("error", "failed to parse token response").
			WithCause(err)
	}

	// Create OAuth token
	token := &auth.OAuthToken{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	return token, nil
}

// GetUserInfo retrieves the user information using the access token
func (p *GoogleProvider) GetUserInfo(ctx context.Context, token *auth.OAuthToken) (auth.AuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", googleUserInfoURL, nil)
	if err != nil {
		return nil, providerErrors.New(ErrAPIRequest).WithCause(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, providerErrors.New(ErrAPIRequest).
			WithDetail("error", err.Error()).
			WithCause(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, providerErrors.New(ErrAPIRequest).
			WithDetail("error", "failed to read response body").
			WithCause(err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, providerErrors.New(ErrInvalidToken)
		}

		return nil, providerErrors.New(ErrAPIRequest).
			WithDetail("status_code", resp.StatusCode).
			WithDetail("body", string(body))
	}

	// Parse user data
	var rawData map[string]interface{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		return nil, providerErrors.New(ErrResponseParsing).
			WithDetail("error", "failed to parse user info").
			WithCause(err)
	}

	// Parse into a struct first for type safety
	var userData struct {
		Sub           string `json:"sub"` // User ID
		Email         string `json:"email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		VerifiedEmail bool   `json:"email_verified"`
		Locale        string `json:"locale"`
		FamilyName    string `json:"family_name"`
		GivenName     string `json:"given_name"`
		Hd            string `json:"hd"`
	}

	if err := json.Unmarshal(body, &userData); err != nil {
		return nil, providerErrors.New(ErrResponseParsing).
			WithDetail("error", "failed to parse user data").
			WithCause(err)
	}

	// Create user info object
	userInfo := &GoogleUserInfo{
		BasicAuthUserInfo: auth.BasicAuthUserInfo{
			ProviderID: userData.Sub,
			Email:      userData.Email,
			Name:       userData.Name,
			Provider:   "google",
			Token:      token,
			RawData:    rawData,
		},
		VerifiedEmail: userData.VerifiedEmail,
		Locale:        userData.Locale,
		FamilyName:    userData.FamilyName,
		GivenName:     userData.GivenName,
		Hd:            userData.Hd,
	}

	// Set optional profile picture
	if userData.Picture != "" {
		picture := userData.Picture
		userInfo.ProfilePicture = &picture
	}

	return userInfo, nil
}

// RefreshToken refreshes an access token using a refresh token
func (p *GoogleProvider) RefreshToken(ctx context.Context, refreshToken string) (*auth.OAuthToken, error) {
	data := url.Values{}
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", googleTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, providerErrors.New(ErrAPIRequest).WithCause(err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, providerErrors.New(ErrAPIRequest).
			WithDetail("error", err.Error()).
			WithCause(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, providerErrors.New(ErrAPIRequest).
			WithDetail("error", "failed to read response body").
			WithCause(err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, providerErrors.New(ErrAPIRequest).
				WithDetail("status_code", resp.StatusCode).
				WithDetail("body", string(body))
		}

		return nil, providerErrors.New(ErrAPIRequest).
			WithDetail("error", errorResp.Error).
			WithDetail("description", errorResp.Description)
	}

	// Parse token response
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, providerErrors.New(ErrResponseParsing).
			WithDetail("error", "failed to parse token response").
			WithCause(err)
	}

	// Create OAuth token - note that refresh_token is typically not included in refresh responses
	// Reuse the original refresh token
	token := &auth.OAuthToken{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: refreshToken, // Reuse the original refresh token
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	return token, nil
}
