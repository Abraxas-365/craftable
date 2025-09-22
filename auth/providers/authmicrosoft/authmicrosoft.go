package authmicrosoft

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/auth"
	"github.com/Abraxas-365/craftable/errx"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// Microsoft OAuth endpoints
	// Using "common" tenant by default - can be overridden
	microsoftAuthURLTemplate  = "https://login.microsoftonline.com/%s/oauth2/v2.0/authorize"
	microsoftTokenURLTemplate = "https://login.microsoftonline.com/%s/oauth2/v2.0/token"
	microsoftUserInfoURL      = "https://graph.microsoft.com/v1.0/me"
	microsoftUserPhotoURL     = "https://graph.microsoft.com/v1.0/me/photo/$value"
	microsoftOrganizationURL  = "https://graph.microsoft.com/v1.0/organization"
	defaultTenant             = "common"
)

// Define Microsoft-specific error codes
var (
	providerErrors = errx.NewRegistry("MICROSOFT_PROVIDER")

	ErrInvalidGrant         = providerErrors.Register("INVALID_GRANT", errx.TypeBadRequest, 400, "Invalid grant")
	ErrInvalidToken         = providerErrors.Register("INVALID_TOKEN", errx.TypeBadRequest, 400, "Invalid token")
	ErrAPIRequest           = providerErrors.Register("API_REQUEST_FAILED", errx.TypeExternal, 500, "Microsoft API request failed")
	ErrResponseParsing      = providerErrors.Register("RESPONSE_PARSING_FAILED", errx.TypeInternal, 500, "Failed to parse Microsoft API response")
	ErrInvalidClient        = providerErrors.Register("INVALID_CLIENT", errx.TypeBadRequest, 400, "Invalid client credentials")
	ErrUnauthorizedClient   = providerErrors.Register("UNAUTHORIZED_CLIENT", errx.TypeAuthorization, 401, "Unauthorized client")
	ErrUnsupportedGrantType = providerErrors.Register("UNSUPPORTED_GRANT_TYPE", errx.TypeBadRequest, 400, "Unsupported grant type")
	ErrInvalidScope         = providerErrors.Register("INVALID_SCOPE", errx.TypeBadRequest, 400, "Invalid scope")
	ErrTokenDecoding        = providerErrors.Register("TOKEN_DECODING_FAILED", errx.TypeInternal, 500, "Failed to decode ID token")
	ErrTenantExtraction     = providerErrors.Register("TENANT_EXTRACTION_FAILED", errx.TypeInternal, 500, "Failed to extract tenant information")
)

// TenantExtractionStrategy defines how to extract tenant information
type TenantExtractionStrategy int

const (
	// TenantFromIDToken extracts tenant from ID token (default, most reliable)
	TenantFromIDToken TenantExtractionStrategy = iota
	// TenantFromOrganizationAPI makes additional API call to get organization info
	TenantFromOrganizationAPI
	// TenantFromBoth tries ID token first, falls back to API call
	TenantFromBoth
	// TenantDisabled disables tenant extraction for performance
	TenantDisabled
)

// MicrosoftUserInfo extends BasicAuthUserInfo with Microsoft-specific fields
type MicrosoftUserInfo struct {
	auth.BasicAuthUserInfo
	GivenName         string   `json:"givenName"`
	Surname           string   `json:"surname"`
	UserPrincipalName string   `json:"userPrincipalName"`
	DisplayName       string   `json:"displayName"`
	JobTitle          *string  `json:"jobTitle"`
	OfficeLocation    *string  `json:"officeLocation"`
	PreferredLanguage *string  `json:"preferredLanguage"`
	MobilePhone       *string  `json:"mobilePhone"`
	BusinessPhones    []string `json:"businessPhones"`
	TenantID          *string  `json:"tenant_id"`
	ObjectID          string   `json:"id"` // Microsoft's unique identifier
}

// MicrosoftProvider implements the OAuthProvider interface for Microsoft
type MicrosoftProvider struct {
	clientID                 string
	clientSecret             string
	redirectURI              string
	tenant                   string // Tenant ID or "common", "organizations", "consumers"
	scopes                   []string
	httpClient               *http.Client
	tenantExtractionStrategy TenantExtractionStrategy
	enableProfilePicture     bool
}

// NewMicrosoftProvider creates a new Microsoft OAuth provider with common tenant
func NewMicrosoftProvider(clientID, clientSecret, redirectURI string) *MicrosoftProvider {
	return &MicrosoftProvider{
		clientID:                 clientID,
		clientSecret:             clientSecret,
		redirectURI:              redirectURI,
		tenant:                   defaultTenant,
		scopes:                   []string{"openid", "profile", "email", "User.Read"},
		httpClient:               &http.Client{Timeout: 15 * time.Second}, // Slightly longer timeout for Microsoft
		tenantExtractionStrategy: TenantFromBoth,                          // Default to most comprehensive strategy
		enableProfilePicture:     true,
	}
}

// NewMicrosoftProviderWithTenant creates a new Microsoft OAuth provider with specific tenant
func NewMicrosoftProviderWithTenant(clientID, clientSecret, redirectURI, tenant string) *MicrosoftProvider {
	return &MicrosoftProvider{
		clientID:                 clientID,
		clientSecret:             clientSecret,
		redirectURI:              redirectURI,
		tenant:                   tenant,
		scopes:                   []string{"openid", "profile", "email", "User.Read"},
		httpClient:               &http.Client{Timeout: 15 * time.Second},
		tenantExtractionStrategy: TenantFromBoth,
		enableProfilePicture:     true,
	}
}

// WithScopes adds additional scopes to the provider
func (p *MicrosoftProvider) WithScopes(scopes ...string) *MicrosoftProvider {
	p.scopes = append(p.scopes, scopes...)
	return p
}

// WithHTTPClient sets a custom HTTP client
func (p *MicrosoftProvider) WithHTTPClient(client *http.Client) *MicrosoftProvider {
	p.httpClient = client
	return p
}

// WithTenant sets a specific tenant for the provider
func (p *MicrosoftProvider) WithTenant(tenant string) *MicrosoftProvider {
	p.tenant = tenant
	return p
}

// WithTenantExtractionStrategy configures how tenant information is extracted
func (p *MicrosoftProvider) WithTenantExtractionStrategy(strategy TenantExtractionStrategy) *MicrosoftProvider {
	p.tenantExtractionStrategy = strategy
	return p
}

// WithProfilePicture enables or disables profile picture retrieval
func (p *MicrosoftProvider) WithProfilePicture(enable bool) *MicrosoftProvider {
	p.enableProfilePicture = enable
	return p
}

// GetAuthURL returns the Microsoft authorization URL
func (p *MicrosoftProvider) GetAuthURL(state string) string {
	authURL := fmt.Sprintf(microsoftAuthURLTemplate, p.tenant)

	params := url.Values{}
	params.Add("client_id", p.clientID)
	params.Add("response_type", "code")
	params.Add("redirect_uri", p.redirectURI)
	params.Add("response_mode", "query")
	params.Add("scope", strings.Join(p.scopes, " "))
	params.Add("state", state)
	params.Add("prompt", "consent") // Force consent to ensure refresh token

	return fmt.Sprintf("%s?%s", authURL, params.Encode())
}

// ExchangeCode exchanges an authorization code for tokens
func (p *MicrosoftProvider) ExchangeCode(ctx context.Context, code string) (*auth.OAuthToken, error) {
	tokenURL := fmt.Sprintf(microsoftTokenURLTemplate, p.tenant)

	data := url.Values{}
	data.Set("client_id", p.clientID)
	data.Set("scope", strings.Join(p.scopes, " "))
	data.Set("code", code)
	data.Set("redirect_uri", p.redirectURI)
	data.Set("grant_type", "authorization_code")
	data.Set("client_secret", p.clientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, providerErrors.New(ErrAPIRequest).WithCause(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

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
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
			ErrorCodes       []int  `json:"error_codes"`
			Timestamp        string `json:"timestamp"`
			TraceID          string `json:"trace_id"`
			CorrelationID    string `json:"correlation_id"`
		}

		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, providerErrors.New(ErrAPIRequest).
				WithDetail("status_code", resp.StatusCode).
				WithDetail("body", string(body))
		}

		// Handle specific Microsoft error types
		switch errorResp.Error {
		case "invalid_grant":
			return nil, providerErrors.New(ErrInvalidGrant).
				WithDetail("error_description", errorResp.ErrorDescription).
				WithDetail("trace_id", errorResp.TraceID)
		case "invalid_client":
			return nil, providerErrors.New(ErrInvalidClient).
				WithDetail("error_description", errorResp.ErrorDescription).
				WithDetail("trace_id", errorResp.TraceID)
		case "unauthorized_client":
			return nil, providerErrors.New(ErrUnauthorizedClient).
				WithDetail("error_description", errorResp.ErrorDescription).
				WithDetail("trace_id", errorResp.TraceID)
		case "unsupported_grant_type":
			return nil, providerErrors.New(ErrUnsupportedGrantType).
				WithDetail("error_description", errorResp.ErrorDescription).
				WithDetail("trace_id", errorResp.TraceID)
		case "invalid_scope":
			return nil, providerErrors.New(ErrInvalidScope).
				WithDetail("error_description", errorResp.ErrorDescription).
				WithDetail("trace_id", errorResp.TraceID)
		default:
			return nil, providerErrors.New(ErrAPIRequest).
				WithDetail("error", errorResp.Error).
				WithDetail("error_description", errorResp.ErrorDescription).
				WithDetail("trace_id", errorResp.TraceID)
		}
	}

	// Parse token response
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
		IDToken      string `json:"id_token"`
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
func (p *MicrosoftProvider) GetUserInfo(ctx context.Context, token *auth.OAuthToken) (auth.AuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", microsoftUserInfoURL, nil)
	if err != nil {
		return nil, providerErrors.New(ErrAPIRequest).WithCause(err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	req.Header.Set("Accept", "application/json")

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

		var errorResp struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}

		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, providerErrors.New(ErrAPIRequest).
				WithDetail("status_code", resp.StatusCode).
				WithDetail("body", string(body))
		}

		return nil, providerErrors.New(ErrAPIRequest).
			WithDetail("error_code", errorResp.Error.Code).
			WithDetail("error_message", errorResp.Error.Message).
			WithDetail("status_code", resp.StatusCode)
	}

	// Parse user data
	var rawData map[string]any
	if err := json.Unmarshal(body, &rawData); err != nil {
		return nil, providerErrors.New(ErrResponseParsing).
			WithDetail("error", "failed to parse user info").
			WithCause(err)
	}

	// Parse into struct for type safety
	var userData struct {
		ID                string   `json:"id"`
		Mail              string   `json:"mail"`
		UserPrincipalName string   `json:"userPrincipalName"`
		DisplayName       string   `json:"displayName"`
		GivenName         string   `json:"givenName"`
		Surname           string   `json:"surname"`
		JobTitle          *string  `json:"jobTitle"`
		OfficeLocation    *string  `json:"officeLocation"`
		PreferredLanguage *string  `json:"preferredLanguage"`
		MobilePhone       *string  `json:"mobilePhone"`
		BusinessPhones    []string `json:"businessPhones"`
	}

	if err := json.Unmarshal(body, &userData); err != nil {
		return nil, providerErrors.New(ErrResponseParsing).
			WithDetail("error", "failed to parse user data").
			WithCause(err)
	}

	// Microsoft sometimes returns empty mail field, fallback to userPrincipalName
	email := userData.Mail
	if email == "" {
		email = userData.UserPrincipalName
	}

	// Construct full name from given name and surname
	name := userData.DisplayName
	if name == "" {
		name = strings.TrimSpace(userData.GivenName + " " + userData.Surname)
	}

	// Create user info object
	userInfo := &MicrosoftUserInfo{
		BasicAuthUserInfo: auth.BasicAuthUserInfo{
			ProviderID: userData.ID,
			Email:      email,
			Name:       name,
			Provider:   "microsoft",
			Token:      token,
			RawData:    rawData,
		},
		GivenName:         userData.GivenName,
		Surname:           userData.Surname,
		UserPrincipalName: userData.UserPrincipalName,
		DisplayName:       userData.DisplayName,
		JobTitle:          userData.JobTitle,
		OfficeLocation:    userData.OfficeLocation,
		PreferredLanguage: userData.PreferredLanguage,
		MobilePhone:       userData.MobilePhone,
		BusinessPhones:    userData.BusinessPhones,
		ObjectID:          userData.ID,
	}

	// Try to get profile picture URL (optional, configurable)
	if p.enableProfilePicture {
		if profilePictureURL := p.getProfilePictureURL(ctx, token.AccessToken); profilePictureURL != "" {
			userInfo.ProfilePicture = &profilePictureURL
		}
	}

	// Extract tenant ID based on configured strategy
	if tenantID := p.extractTenantFromToken(ctx, token, rawData); tenantID != "" {
		userInfo.TenantID = &tenantID
	}

	return userInfo, nil
}

// getProfilePictureURL attempts to get the user's profile picture URL
// This is optional and non-blocking - if it fails, we continue without the picture
func (p *MicrosoftProvider) getProfilePictureURL(ctx context.Context, accessToken string) string {
	// Create a context with a short timeout for this optional operation
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", microsoftUserPhotoURL, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	// If successful, Microsoft returns the actual image data
	// For this implementation, we'll return the API URL as the profile picture URL
	// In a production system, you might want to download and store the image
	if resp.StatusCode == http.StatusOK {
		return microsoftUserPhotoURL
	}

	return ""
}

// extractTenantFromToken extracts tenant information based on configured strategy
func (p *MicrosoftProvider) extractTenantFromToken(ctx context.Context, token *auth.OAuthToken, rawData map[string]any) string {
	if p.tenantExtractionStrategy == TenantDisabled {
		return ""
	}

	var tenantID string

	// Strategy 1: Try to extract from stored ID token in raw data
	if p.tenantExtractionStrategy == TenantFromIDToken || p.tenantExtractionStrategy == TenantFromBoth {
		if idToken := p.getIDTokenFromRawData(rawData); idToken != "" {
			if tid := p.ExtractTenantFromIDToken(idToken); tid != "" {
				tenantID = tid
			}
		}
	}

	// Strategy 2: Fallback to API call if needed and configured
	if tenantID == "" && (p.tenantExtractionStrategy == TenantFromOrganizationAPI || p.tenantExtractionStrategy == TenantFromBoth) {
		if tid := p.extractTenantFromOrganizationAPI(ctx, token.AccessToken); tid != "" {
			tenantID = tid
		}
	}

	return tenantID
}

// ExtractTenantFromIDToken is a public method that allows users to extract tenant ID from an ID token
// This is useful when users handle the token exchange themselves and want to extract tenant information
func (p *MicrosoftProvider) ExtractTenantFromIDToken(idToken string) string {
	return p.extractTenantFromIDToken(idToken)
}

// ExtractTenantFromOrganizationAPI is a public method that allows users to extract tenant ID via API call
// This requires the Organization.Read.All scope or User.Read scope
func (p *MicrosoftProvider) ExtractTenantFromOrganizationAPI(ctx context.Context, accessToken string) string {
	return p.extractTenantFromOrganizationAPI(ctx, accessToken)
}

// GetTenantIDFromTokenResponse is a helper method for users to extract tenant from full token response
// Users can call this method immediately after token exchange to get tenant information
func (p *MicrosoftProvider) GetTenantIDFromTokenResponse(ctx context.Context, tokenResponse map[string]any, accessToken string) string {
	// First try ID token if available
	if idToken, ok := tokenResponse["id_token"].(string); ok && idToken != "" {
		if tenantID := p.ExtractTenantFromIDToken(idToken); tenantID != "" {
			return tenantID
		}
	}

	// Fallback to API call
	return p.ExtractTenantFromOrganizationAPI(ctx, accessToken)
}

// getIDTokenFromRawData extracts the ID token from stored raw data or token response
func (p *MicrosoftProvider) getIDTokenFromRawData(rawData map[string]any) string {
	// Check if ID token is stored in raw data (from token exchange)
	if idToken, ok := rawData["id_token"].(string); ok && idToken != "" {
		return idToken
	}

	// Check for token response data stored in raw data
	if tokenData, ok := rawData["token_response"]; ok {
		if tokenMap, ok := tokenData.(map[string]any); ok {
			if idToken, ok := tokenMap["id_token"].(string); ok && idToken != "" {
				return idToken
			}
		}
	}

	return ""
}

// extractTenantFromIDToken decodes the ID token and extracts tenant ID
func (p *MicrosoftProvider) extractTenantFromIDToken(idToken string) string {
	// Parse the JWT token without verification (we only need claims)
	// Microsoft's ID tokens are signed and we trust them since they came from Microsoft
	token, _, err := new(jwt.Parser).ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return ""
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		// Extract tenant ID from 'tid' claim
		if tid, ok := claims["tid"].(string); ok && tid != "" {
			return tid
		}

		// Fallback: extract from issuer URL
		if iss, ok := claims["iss"].(string); ok && iss != "" {
			return p.extractTenantFromIssuer(iss)
		}
	}

	return ""
}

// extractTenantFromIssuer extracts tenant ID from issuer URL
func (p *MicrosoftProvider) extractTenantFromIssuer(issuer string) string {
	// Microsoft issuer format: https://login.microsoftonline.com/{tenant}/v2.0
	parts := strings.Split(issuer, "/")
	if len(parts) >= 4 && strings.Contains(issuer, "login.microsoftonline.com") {
		tenantPart := parts[3]
		// Validate it's not a generic endpoint
		if tenantPart != "common" && tenantPart != "organizations" && tenantPart != "consumers" && tenantPart != "v2.0" {
			return tenantPart
		}
	}
	return ""
}

// extractTenantFromOrganizationAPI makes an API call to get organization information
func (p *MicrosoftProvider) extractTenantFromOrganizationAPI(ctx context.Context, accessToken string) string {
	// Create a context with timeout for this operation
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", microsoftOrganizationURL, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var orgResp struct {
		Value []struct {
			ID string `json:"id"`
		} `json:"value"`
	}

	if err := json.Unmarshal(body, &orgResp); err != nil {
		return ""
	}

	// Return the first organization's ID (tenant ID)
	if len(orgResp.Value) > 0 && orgResp.Value[0].ID != "" {
		return orgResp.Value[0].ID
	}

	return ""
}

// RefreshToken refreshes an access token using a refresh token
func (p *MicrosoftProvider) RefreshToken(ctx context.Context, refreshToken string) (*auth.OAuthToken, error) {
	tokenURL := fmt.Sprintf(microsoftTokenURLTemplate, p.tenant)

	data := url.Values{}
	data.Set("client_id", p.clientID)
	data.Set("scope", strings.Join(p.scopes, " "))
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")
	data.Set("client_secret", p.clientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, providerErrors.New(ErrAPIRequest).WithCause(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

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
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
			ErrorCodes       []int  `json:"error_codes"`
			Timestamp        string `json:"timestamp"`
			TraceID          string `json:"trace_id"`
			CorrelationID    string `json:"correlation_id"`
		}

		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, providerErrors.New(ErrAPIRequest).
				WithDetail("status_code", resp.StatusCode).
				WithDetail("body", string(body))
		}

		return nil, providerErrors.New(ErrAPIRequest).
			WithDetail("error", errorResp.Error).
			WithDetail("error_description", errorResp.ErrorDescription).
			WithDetail("trace_id", errorResp.TraceID)
	}

	// Parse token response
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
		IDToken      string `json:"id_token"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, providerErrors.New(ErrResponseParsing).
			WithDetail("error", "failed to parse token response").
			WithCause(err)
	}

	// Create OAuth token
	// Microsoft may or may not return a new refresh token
	// If no new refresh token is provided, reuse the original one
	newRefreshToken := tokenResp.RefreshToken
	if newRefreshToken == "" {
		newRefreshToken = refreshToken
	}

	token := &auth.OAuthToken{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	return token, nil
}
