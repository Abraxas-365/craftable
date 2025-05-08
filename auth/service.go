package auth

import (
	"context"
	"errors"
	"time"

	"github.com/Conversia-AI/craftale/errx"
	"github.com/golang-jwt/jwt/v5"
)

// Define error codes/types
var (
	// ErrorRegistry for auth package
	authErrors = errx.NewRegistry("AUTH")

	// Error code definitions
	ErrProviderNotFound     = authErrors.Register("PROVIDER_NOT_FOUND", errx.TypeNotFound, 404, "OAuth provider not found")
	ErrCodeExchange         = authErrors.Register("CODE_EXCHANGE_FAILED", errx.TypeBadRequest, 400, "Failed to exchange code for token")
	ErrUserInfo             = authErrors.Register("USER_INFO_FAILED", errx.TypeInternal, 500, "Failed to get user info from provider")
	ErrUserCreation         = authErrors.Register("USER_CREATION_FAILED", errx.TypeInternal, 500, "Failed to create user")
	ErrUserDisabled         = authErrors.Register("USER_DISABLED", errx.TypeAuthorization, 403, "User account is disabled")
	ErrOAuthAccountCreation = authErrors.Register("OAUTH_ACCOUNT_CREATION_FAILED", errx.TypeInternal, 500, "Failed to create OAuth account")
	ErrTokenGeneration      = authErrors.Register("TOKEN_GENERATION_FAILED", errx.TypeInternal, 500, "Failed to generate JWT token")
	ErrUserNotFound         = authErrors.Register("USER_NOT_FOUND", errx.TypeNotFound, 404, "User not found")
	ErrInvalidToken         = authErrors.Register("INVALID_TOKEN", errx.TypeAuthorization, 401, "Invalid or expired token")
)

// IsUserNotFound helper function
func IsUserNotFound(err error) bool {
	return errx.IsCode(err, ErrUserNotFound)
}

// Service implementation
type service struct {
	providers       map[string]OAuthProvider
	userStore       UserStore
	oauthStore      OAuthAccountStore
	jwtSecret       []byte
	tokenExpiration time.Duration
}

// NewAuthService creates a new auth service
func NewAuthService(
	userStore UserStore,
	oauthStore OAuthAccountStore,
	jwtSecret []byte,
	tokenExpiration time.Duration,
) Service {
	return &service{
		providers:       make(map[string]OAuthProvider),
		userStore:       userStore,
		oauthStore:      oauthStore,
		jwtSecret:       jwtSecret,
		tokenExpiration: tokenExpiration,
	}
}

// GetAuthURL returns the authorization URL for the specified provider
func (s *service) GetAuthURL(provider, state string) (string, error) {
	p, ok := s.providers[provider]
	if !ok {
		return "", authErrors.New(ErrProviderNotFound).WithDetail("provider", provider)
	}
	return p.GetAuthURL(state), nil
}

// HandleOAuthCallback processes the OAuth callback and returns the authenticated user
func (s *service) HandleOAuthCallback(ctx context.Context, provider, code string) (*AuthResponse, error) {
	p, ok := s.providers[provider]
	if !ok {
		return nil, authErrors.New(ErrProviderNotFound).WithDetail("provider", provider)
	}

	// Exchange code for token
	token, err := p.ExchangeCode(ctx, code)
	if err != nil {
		return nil, authErrors.New(ErrCodeExchange).
			WithDetail("provider", provider).
			WithCause(err)
	}

	// Get user info from provider
	userInfo, err := p.GetUserInfo(ctx, token)
	if err != nil {
		return nil, authErrors.New(ErrUserInfo).
			WithDetail("provider", provider).
			WithCause(err)
	}

	// Check if user exists by provider ID
	user, err := s.userStore.GetUserByProviderID(ctx, provider, userInfo.GetProviderID())
	if err != nil {
		if IsUserNotFound(err) {
			// User doesn't exist, create new user
			user, err = s.userStore.CreateUser(ctx, userInfo)
			if err != nil {
				return nil, authErrors.New(ErrUserCreation).
					WithDetail("email", userInfo.GetEmail()).
					WithCause(err)
			}
		} else {
			return nil, authErrors.New(ErrUserInfo).WithCause(err)
		}

		// Check if user is nil
		if user == nil {
			return nil, authErrors.New(ErrUserCreation).
				WithDetail("message", "User creation returned nil user")
		}
	}

	// Check if user is enabled
	if !user.IsActive() {
		return nil, authErrors.New(ErrUserDisabled).
			WithDetail("user_id", user.GetID())
	}

	// Create or update OAuth account
	err = s.oauthStore.CreateOAuthAccount(ctx, user.GetID(), userInfo)
	if err != nil {
		return nil, authErrors.New(ErrOAuthAccountCreation).
			WithDetail("user_id", user.GetID()).
			WithDetail("provider", provider).
			WithCause(err)
	}

	// Generate JWT token
	tokenString, err := s.GenerateToken(user)
	if err != nil {
		return nil, authErrors.New(ErrTokenGeneration).
			WithDetail("user_id", user.GetID()).
			WithCause(err)
	}

	return &AuthResponse{
		User:        user,
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.tokenExpiration.Seconds()),
	}, nil
}

func (s *service) GenerateToken(user User) (string, error) {
	now := time.Now()
	claims := &JWTClaims{ // Note the & to create a pointer
		UserID:    user.GetID(),
		Email:     user.GetEmail(),
		IssuedAt:  now,
		ExpiresAt: now.Add(s.tokenExpiration),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", authErrors.New(ErrTokenGeneration).
			WithDetail("error", err.Error()).
			WithCause(err)
	}

	return tokenString, nil
}

// ValidateToken verifies a JWT token and returns the claims
func (s *service) ValidateToken(tokenString string) (*JWTClaims, error) {
	claims := &JWTClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, authErrors.New(ErrInvalidToken).
				WithDetail("error", "invalid signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		// Handle JWT errors - jwt/v5 uses wrapped errors
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, authErrors.New(ErrInvalidToken).
				WithDetail("error", "token expired").
				WithCause(err)
		} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, authErrors.New(ErrInvalidToken).
				WithDetail("error", "token not valid yet").
				WithCause(err)
		} else if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, authErrors.New(ErrInvalidToken).
				WithDetail("error", "token malformed").
				WithCause(err)
		} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, authErrors.New(ErrInvalidToken).
				WithDetail("error", "invalid signature").
				WithCause(err)
		} else {
			return nil, authErrors.New(ErrInvalidToken).
				WithDetail("error", err.Error()).
				WithCause(err)
		}
	}

	if !token.Valid {
		return nil, authErrors.New(ErrInvalidToken).
			WithDetail("error", "token validation failed")
	}

	return claims, nil
}

// RegisterProvider adds a new OAuth provider to the service
func (s *service) RegisterProvider(name string, provider OAuthProvider) {
	s.providers[name] = provider
}
