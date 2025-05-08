# Craftable

<p align="center">
  <img src="https://static.wikia.nocookie.net/minecraft_gamepedia/images/b/b7/Crafting_Table_JE4_BE3.png/revision/latest?cb=20191229083528" alt="Craftable Logo" width="200" height="200">
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/Abraxas-365/craftable"><img src="https://pkg.go.dev/badge/github.com/Abraxas-365/craftable.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/Abraxas-365/craftable"><img src="https://goreportcard.com/badge/github.com/Abraxas-365/craftable" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/Abraxas-365/craftable" alt="License"></a>
  <a href="https://github.com/Abraxas-365/craftable/releases"><img src="https://img.shields.io/github/v/release/Abraxas-365/craftable" alt="GitHub release"></a>
</p>

**Craftable** is a collection of high-quality, reusable Go packages designed to accelerate application development. It provides elegant solutions for common challenges like error handling, authentication, and CLI interactions.

## 📦 Packages

### errx - Extended Error Handling

A robust error package that makes error handling more structured and informative:

- **Structured Errors**: Type, code, and detailed context for every error
- **Error Registry**: Domain-specific error definitions with prefixes
- **Beautiful CLI Errors**: Multiple display modes from simple to detailed
- **Web Framework Integration**: Works with standard net/http and Fiber
- **Error Wrapping**: Preserves error causes while adding context

### auth - Flexible Authentication

An interface-based authentication system that adapts to any project:

- **OAuth Integration**: Support for multiple providers (Google, etc.)
- **JWT Authentication**: Token generation and validation
- **Interface-Based Design**: Works with any user model
- **Provider Flexibility**: Extensible for any OAuth provider
- **Secure by Default**: Implements authentication best practices

## 🚀 Installation

```bash
go get github.com/Abraxas-365/craftable
```

Or install specific packages:

```bash
go get github.com/Abraxas-365/craftable/errx
go get github.com/Abraxas-365/craftable/auth
```

## 🔍 Example Usage

### Error Handling (errx)

```go
package main

import (
    "net/http"
    
    "github.com/Abraxas-365/craftable/errx"
)

func main() {
    // Create an error registry
    userErrors := errx.NewRegistry("USER")
    
    // Register common error types
    ErrUserNotFound := userErrors.Register("NOT_FOUND", errx.TypeNotFound, 
        http.StatusNotFound, "User not found")
    
    // Use in your application
    if userNotFound {
        return userErrors.New(ErrUserNotFound).
            WithDetail("user_id", "123").
            WithDetail("request_id", requestID)
    }
}
```

### Authentication (auth)

```go
package main

import (
    "time"
    
    "github.com/Abraxas-365/craftable/auth"
    "github.com/Abraxas-365/craftable/auth/providers"
)

func main() {
    // Create your store implementations
    userStore := NewYourUserStore()
    oauthStore := NewYourOAuthStore()
    
    // Create auth service
    authService := auth.NewAuthService(
        userStore,
        oauthStore,
        []byte("your-jwt-secret"),
        24*time.Hour,
    )
    
    // Register providers
    googleProvider := providers.NewGoogleProvider(
        "client-id", 
        "client-secret", 
        "https://your-app.com/auth/callback/google",
    )
    
    authService.RegisterProvider("google", googleProvider)
}
```

## 📐 Design Principles

Craftable follows these core principles:

1. **Interface-Based Design**: Flexible abstractions that adapt to different projects
2. **Detailed Error Handling**: Errors provide rich context for debugging and user feedback
3. **Minimal Dependencies**: Focused packages with few external requirements
4. **Beautiful User Experience**: Whether CLI or API, interactions are elegant and informative
5. **Production Ready**: Built for real-world applications, not just examples

## 📚 Documentation

For detailed documentation and examples for each package, see:

- [errx Documentation](https://pkg.go.dev/github.com/Abraxas-365/craftable/errx)
- [auth Documentation](https://pkg.go.dev/github.com/Abraxas-365/craftable/auth)

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
