package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Core OAuth token structure
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// OAuthAccount represents the stored OAuth credentials
type OAuthAccount struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Provider      string         `json:"provider"`
	ProviderID    string         `json:"provider_id"`
	ProviderEmail string         `json:"provider_email"`
	AccessToken   string         `json:"access_token"`
	RefreshToken  string         `json:"refresh_token"`
	ExpiresAt     time.Time      `json:"expires_at"`
	Metadata      map[string]any `json:"metadata"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// AuthUserInfo as an interface allowing different implementations
type AuthUserInfo interface {
	GetProviderID() string
	GetEmail() string
	GetName() string
	GetProvider() string
	GetProfilePicture() *string
	GetToken() *OAuthToken
	GetRawData() map[string]any // Access to the raw data from provider
}

// BasicAuthUserInfo is a standard implementation of AuthUserInfo
type BasicAuthUserInfo struct {
	ProviderID     string         `json:"provider_id"`
	Email          string         `json:"email"`
	Name           string         `json:"name"`
	Provider       string         `json:"provider"`
	ProfilePicture *string        `json:"profile_picture,omitempty"`
	Token          *OAuthToken    `json:"token"`
	RawData        map[string]any `json:"raw_data,omitempty"`
}

// Implement AuthUserInfo interface
func (u *BasicAuthUserInfo) GetProviderID() string      { return u.ProviderID }
func (u *BasicAuthUserInfo) GetEmail() string           { return u.Email }
func (u *BasicAuthUserInfo) GetName() string            { return u.Name }
func (u *BasicAuthUserInfo) GetProvider() string        { return u.Provider }
func (u *BasicAuthUserInfo) GetProfilePicture() *string { return u.ProfilePicture }
func (u *BasicAuthUserInfo) GetToken() *OAuthToken      { return u.Token }
func (u *BasicAuthUserInfo) GetRawData() map[string]any { return u.RawData }

// User represents the basic requirements for a user model
type User interface {
	GetID() string
	GetEmail() string
	IsActive() bool
}

// AuthResponse contains authentication result
type AuthResponse struct {
	User        User   `json:"user"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// JWTClaims for token generation
type JWTClaims struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"exp"`
	IssuedAt  time.Time `json:"iat"`
}

// Implement jwt.Claims interface methods
func (c *JWTClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(c.ExpiresAt), nil
}

func (c *JWTClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(c.IssuedAt), nil
}

func (c *JWTClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(c.IssuedAt), nil // Using IssuedAt as NotBefore
}

func (c *JWTClaims) GetIssuer() (string, error) {
	return "", nil // Not using issuer
}

func (c *JWTClaims) GetSubject() (string, error) {
	return c.UserID, nil // Subject is the user ID
}

func (c *JWTClaims) GetAudience() (jwt.ClaimStrings, error) {
	return nil, nil // Not using audience
}

// OAuthProvider interface
type OAuthProvider interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*OAuthToken, error)
	GetUserInfo(ctx context.Context, token *OAuthToken) (AuthUserInfo, error)
	RefreshToken(ctx context.Context, refreshToken string) (*OAuthToken, error)
}

// UserStore interface
type UserStore interface {
	CreateUser(ctx context.Context, userInfo AuthUserInfo) (User, error)
	GetUserByProviderID(ctx context.Context, provider, providerID string) (User, error)
}

// OAuthAccountStore interface
type OAuthAccountStore interface {
	CreateOAuthAccount(ctx context.Context, userID string, info AuthUserInfo) error
	GetOAuthAccount(ctx context.Context, provider, providerID string) (*OAuthAccount, error)
	UpdateOAuthToken(ctx context.Context, provider, providerID string, token *OAuthToken) error
}

// Service interface
type Service interface {
	GetAuthURL(provider, state string) (string, error)
	HandleOAuthCallback(ctx context.Context, provider, code string) (*AuthResponse, error)
	RegisterProvider(name string, provider OAuthProvider)
	GenerateToken(user User) (string, error)
	ValidateToken(tokenString string) (*JWTClaims, error)
}
