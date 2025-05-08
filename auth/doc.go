/*
Package auth provides a flexible, interface-based authentication system for Go applications.
It supports OAuth 2.0 authentication flows with various providers and includes JWT-based
token generation and validation.

# Core Concepts

The auth package is built around several key interfaces:

  - User: Represents an authenticated user
  - AuthUserInfo: Information returned from OAuth providers
  - OAuthProvider: Abstraction for OAuth providers like Google, GitHub, etc.
  - UserStore: Storage interface for user data
  - OAuthAccountStore: Storage interface for OAuth account information
  - Service: Main auth service that orchestrates the authentication process

# Basic Usage

Initialize the auth service with your storage implementations and JWT configuration:

	// Create your storage implementations
	userStore := YourUserStoreImplementation{}
	oauthStore := YourOAuthStoreImplementation{}

	// Create auth service
	authService := auth.NewAuthService(
		userStore,
		oauthStore,
		[]byte("your-jwt-secret"),
		24*time.Hour, // Token expiration
	)

	// Register OAuth providers
	googleProvider := providers.NewGoogleProvider(
		"google-client-id",
		"google-client-secret",
		"https://your-app.com/auth/callback/google",
	)
	authService.RegisterProvider("google", googleProvider)

# OAuth Authentication Flow

The typical OAuth flow with this package:

1. Generate an authorization URL:

	authURL, err := authService.GetAuthURL("google", randomState)
	// Redirect user to authURL

2. Handle callback after user authorizes:

	authResponse, err := authService.HandleOAuthCallback(ctx, "google", code)
	if err != nil {
		// Handle authentication error
	}

	// Authentication successful
	user := authResponse.User
	token := authResponse.AccessToken

3. Validate tokens in protected routes:

	claims, err := authService.ValidateToken(tokenFromRequest)
	if err != nil {
		// Token invalid or expired
	}

	// Token valid, claims.UserID contains the authenticated user ID

# Implementing the Interfaces

To use this package, you need to implement several interfaces:

User Interface:

	type CustomUser struct {
		ID       string
		Email    string
		IsActive bool
	}

	func (u *CustomUser) GetID() string     { return u.ID }
	func (u *CustomUser) GetEmail() string  { return u.Email }
	func (u *CustomUser) IsActive() bool    { return u.IsActive }

Storage Interfaces:

	func (s *MyUserStore) CreateUser(ctx context.Context, userInfo auth.AuthUserInfo) (auth.User, error) {
		// Implementation...
	}

	func (s *MyUserStore) GetUserByProviderID(ctx context.Context, provider, providerID string) (auth.User, error) {
		// Implementation...
	}

# Extending the Package

Add more OAuth providers by implementing the OAuthProvider interface:

	type CustomProvider struct {
		// Provider-specific fields
	}

	func (p *CustomProvider) GetAuthURL(state string) string {
		// Implementation...
	}

	func (p *CustomProvider) ExchangeCode(ctx context.Context, code string) (*auth.OAuthToken, error) {
		// Implementation...
	}

	// Implement other required methods...

# Error Handling

The package uses the errx package for standardized error handling:

	if err != nil {
		if auth.IsUserNotFound(err) {
			// Handle user not found case
		}

		// Check for other specific error types
		if errx.IsType(err, errx.TypeAuthorization) {
			// Handle authorization errors
		}
	}

# Provider-Specific Extensions

Providers can return extended user information through the AuthUserInfo interface:

	googleUser, ok := userInfo.(*providers.GoogleUserInfo)
	if ok {
		// Access Google-specific fields
		domain := googleUser.Hd  // G Suite domain
	}
*/
package auth
