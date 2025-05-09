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


## Table of Contents

- [📚 Craftable Library](#-craftable-library)
  - [Table of Contents](#table-of-contents)
  - [📦 Packages](#-packages)
    - [errx - Extended Error Handling](#errx---extended-error-handling)
    - [auth - Flexible Authentication](#auth---flexible-authentication)
    - [storex - Database Store Abstraction](#storex---database-store-abstraction)
    - [dtox - DTO/Model Conversion](#dtox---dtomodel-conversion)
    - [validatex - Struct Validation](#validatex---struct-validation)
    - [ai - Artificial Intelligence Toolkit](#ai---artificial-intelligence-toolkit)
      - [llm - Large Language Model Client](#llm---large-language-model-client)
      - [embedding - Text Embedding Interface](#embedding---text-embedding-interface)
      - [ocr - Optical Character Recognition](#ocr---optical-character-recognition)
  - [🚀 Installation](#-installation)
  - [📝 Example Usage](#-example-usage)
    - [Error Handling (errx)](#error-handling-errx)
    - [Authentication (auth)](#authentication-auth)
    - [Database Store (storex)](#database-store-storex)
      - [Basic CRUD Operations](#basic-crud-operations)
      - [Pagination and Filtering](#pagination-and-filtering)
    - [DTO/Model Conversion (dtox)](#dtomodel-conversion-dtox)
    - [Struct Validation (validatex)](#struct-validation-validatex)
    - [LLM Interaction](#llm-interaction)
    - [Text Embedding](#text-embedding)
    - [OCR Processing](#ocr-processing)
  - [🎨 Design Principles](#-design-principles)
  - [📚 Documentation](#-documentation)
  - [📜 License](#-license)
  - [🤝 Contributing](#-contributing)

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

### storex - Database Store Abstraction

A generic abstraction layer for working with different database stores:

- **Type-safe Generic Implementations**: Strongly-typed database operations
- **Complete CRUD Operations**: Unified API for standard operations
- **Advanced Pagination**: Consistent pagination with sorting and filtering
- **Bulk Operations**: Efficient batch processing for large datasets
- **Transaction Support**: Safe database operations with rollback capability
- **Query Builder**: Type-safe fluent interface for complex queries
- **Full-text Search**: Powerful search capabilities for MongoDB and SQL
- **Change Notifications**: Real-time data change streams (MongoDB)
- **Consistent Error Handling**: Detailed context for database errors

### dtox - DTO/Model Conversion

A generic package for type-safe conversion between DTOs and domain models:

- **Type-safe Mappings**: Strongly-typed conversion between DTOs and domain models
- **Field Mapping**: Flexible field name matching between different structures
- **Validation**: Integrated validation with detailed error reporting
- **Batch Operations**: Efficient processing of collections with parallel support
- **Custom Conversion**: Support for custom conversion functions
- **Partial Updates**: Simplified handling of partial object updates
- **Reflection-Based**: Automatic field discovery and mapping

### validatex - Struct Validation

A flexible validation system using struct tags with structured error handling:

- **Tag-based Validation**: Define validation rules directly in struct tags
- **Rich Validation Rules**: Support for required fields, length/value constraints, patterns, etc.
- **Structured Errors**: Detailed validation errors with field context
- **Nested Validation**: Support for validating nested structs and slices
- **Custom Validators**: Easily extend with custom validation functions
- **Integration with errx**: Consistent error handling throughout your application

### ai - Artificial Intelligence Toolkit

A collection of packages for working with AI services and capabilities:

#### llm - Large Language Model Client

Interact with large language models with a clean, flexible interface:

- **Multiple Providers**: Support for various LLM providers (OpenAI, etc.)
- **Streaming Support**: Stream responses for real-time interactions
- **Message Management**: Structured conversation history handling
- **Tool Integration**: Support for function calling and tool usage
- **Type-safe Responses**: Strongly-typed message handling

#### embedding - Text Embedding Interface

Convert text into vector embeddings for semantic understanding:

- **Document Processing**: Batch processing of document collections
- **Query Embedding**: Specialized handling for query text
- **Flexible Models**: Support for different embedding models and dimensions
- **Usage Tracking**: Token and request usage monitoring
- **Provider Abstraction**: Common interface across embedding providers

#### ocr - Optical Character Recognition

Extract text from images with confidence scores and structured results:

- **Image Processing**: Process images from files or URLs
- **Confidence Scoring**: Detailed confidence metrics for extracted text
- **Text Blocks**: Structured information about detected text regions
- **Multiple Languages**: Support for various languages and scripts
- **Format Flexibility**: Works with different image formats

## 🚀 Installation

```bash
go get github.com/Abraxas-365/craftable
```

Or install specific packages:

```bash
go get github.com/Abraxas-365/craftable/errx
go get github.com/Abraxas-365/craftable/auth
go get github.com/Abraxas-365/craftable/storex
go get github.com/Abraxas-365/craftable/dtox
go get github.com/Abraxas-365/craftable/validatex
go get github.com/Abraxas-365/craftable/ai/llm
go get github.com/Abraxas-365/craftable/ai/embedding
go get github.com/Abraxas-365/craftable/ai/ocr
```

## 📝 Example Usage

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

### Database Store (storex)

#### Basic CRUD Operations

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/Abraxas-365/craftable/storex"
    "go.mongodb.org/mongo-driver/mongo"
)

// Define your model
type User struct {
    ID        string `bson:"_id,omitempty"`
    Name      string `bson:"name"`
    Email     string `bson:"email"`
    CreatedAt int64  `bson:"created_at"`
}

func ExampleCRUD(client *mongo.Client) {
    // Get collection
    collection := client.Database("myapp").Collection("users")

    // Create a typed store for Users
    userStore := storex.NewTypedMongo[User](collection)
    ctx := context.Background()
    
    // Create a new user
    newUser := User{
        Name:      "John Doe",
        Email:     "john@example.com",
        CreatedAt: time.Now().Unix(),
    }
    
    createdUser, err := userStore.Create(ctx, newUser)
    if err != nil {
        // Handle error
        return
    }
    
    // Find by ID
    user, err := userStore.FindByID(ctx, createdUser.ID)
    if err != nil {
        // Handle error
        return
    }
    
    // Update user
    user.Name = "John Smith"
    updatedUser, err := userStore.Update(ctx, user.ID, user)
    if err != nil {
        // Handle error
        return
    }
    
    // Delete user
    err = userStore.Delete(ctx, user.ID)
    if err != nil {
        // Handle error
        return
    }
}
```

#### Pagination and Filtering

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/Abraxas-365/craftable/storex"
    "go.mongodb.org/mongo-driver/mongo"
)

func ExamplePagination(client *mongo.Client) {
    collection := client.Database("myapp").Collection("users")
    userStore := storex.NewTypedMongo[User](collection)

    // Create pagination options
    opts := storex.DefaultPaginationOptions()
    opts.Page = 1
    opts.PageSize = 10
    opts.OrderBy = "created_at"
    opts.Desc = true
    
    // Add filters
    opts = opts.WithFilter("name", "John")
    opts = opts.WithFilter("active", true)

    // Perform paginated query
    ctx := context.Background()
    result, err := userStore.Paginate(ctx, opts)
    if err != nil {
        // Handle error
        return
    }

    // Access results
    for _, user := range result.Data {
        fmt.Println(user.Name, user.Email)
    }

    // Pagination metadata
    fmt.Printf("Page %d of %d (Total: %d items)\n", 
        result.Page.Number, result.Page.Pages, result.Page.Total)
    
    // Check for more pages
    if result.HasNext() {
        fmt.Println("More pages available")
    }
}
```

### DTO/Model Conversion (dtox)

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/Abraxas-365/craftable/dtox"
    "github.com/Abraxas-365/craftable/errx"
)

// Define DTO and model types
type UserDTO struct {
    FullName string `json:"full_name"`
    Email    string `json:"email"`
    Age      int    `json:"age"`
}

type User struct {
    ID        string
    FirstName string
    LastName  string
    Email     string
    Age       int
    CreatedAt time.Time
}

func main() {
    // Create a mapper with field mapping
    mapper := dtox.NewMapper[UserDTO, User]().
        WithFieldMapping("FullName", "FirstName").
        WithValidation(func(dto UserDTO) error {
            if dto.Age < 18 {
                return fmt.Errorf("user must be at least 18 years old")
            }
            return nil
        })
    
    // Convert DTO to model
    dto := UserDTO{
        FullName: "John Doe",
        Email:    "john@example.com",
        Age:      30,
    }
    
    user, err := mapper.ToModel(dto)
    if err != nil {
        fmt.Printf("Error: %s\n", err)
        return
    }
    
    // The model now has FirstName = "John Doe"
    fmt.Printf("User: %+v\n", user)
    
    // Convert multiple DTOs to models
    dtos := []UserDTO{
        {FullName: "Jane Smith", Email: "jane@example.com", Age: 28},
        {FullName: "Bob Johnson", Email: "bob@example.com", Age: 35},
    }
    
    users, err := mapper.ToModels(dtos)
    if err != nil {
        fmt.Printf("Error: %s\n", err)
        return
    }
    
    fmt.Printf("Converted %d users\n", len(users))
    
    // Handle partial updates
    partialMapper := dtox.NewMapper[UserDTO, User]().WithPartial(true)
    
    // Only fields that are non-zero in the DTO will be updated
    partialDto := UserDTO{
        Email: "newemail@example.com",
        // Age and FullName are not set, so they won't be updated
    }
    
    updatedUser, err := partialMapper.ApplyPartialUpdate(user, partialDto)
    if err != nil {
        fmt.Printf("Error: %s\n", err)
        return
    }
    
    // Email has been updated, but FirstName and Age remain unchanged
    fmt.Printf("Updated user: %+v\n", updatedUser)
}
```

### Struct Validation (validatex)

```go
package main

import (
    "fmt"
    "net/http"
    
    "github.com/Abraxas-365/craftable/errx"
    "github.com/Abraxas-365/craftable/validatex"
)

// Define a struct with validation rules
type CreateUserRequest struct {
    Username  string `json:"username" validatex:"required,min=3,max=50"`
    Email     string `json:"email" validatex:"required,email"`
    Age       int    `json:"age" validatex:"min=18"`
    Password  string `json:"password" validatex:"required,min=8"`
    Role      string `json:"role" validatex:"oneof=admin user guest"`
}

// Custom validation function for strong passwords
func init() {
    validatex.RegisterValidationFunc("strongpassword", func(value interface{}, param string) bool {
        password, ok := value.(string)
        if !ok {
            return false
        }
        
        // Check for minimum length, uppercase, lowercase, number, and special character
        if len(password) < 8 {
            return false
        }
        
        hasUpper := false
        hasLower := false
        hasNumber := false
        hasSpecial := false
        
        for _, char := range password {
            switch {
            case 'A' <= char && char <= 'Z':
                hasUpper = true
            case 'a' <= char && char <= 'z':
                hasLower = true
            case '0' <= char && char <= '9':
                hasNumber = true
            case char == '!' || char == '@' || char == '#' || char == '$' || char == '%':
                hasSpecial = true
            }
        }
        
        return hasUpper && hasLower && hasNumber && hasSpecial
    })
}

func HandleCreateUser(w http.ResponseWriter, r *http.Request) {
    // Parse request body
    var req CreateUserRequest
    decoder := json.NewDecoder(r.Body)
    if err := decoder.Decode(&req); err != nil {
        errx.New("Invalid JSON payload", errx.TypeBadRequest).
            WithHTTPStatus(http.StatusBadRequest).
            ToHTTP(w)
        return
    }
    
    // Validate the request
    if err := validatex.ValidateWithErrx(req); err != nil {
        // err is an errx.Error with detailed validation information
        err.ToHTTP(w)
        return
    }
    
    // Request is valid, proceed with user creation
    // ...
    
    // Return success response
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{
        "status":  "success",
        "message": "User created successfully",
    })
}

func main() {
    // Example validation
    user := CreateUserRequest{
        Username: "jo", // too short
        Email:    "not-an-email",
        Age:      15,   // too young
        Password: "weak",
        Role:     "superuser", // not in allowed values
    }
    
    if err := validatex.ValidateWithErrx(user); err != nil {
        // Structured error with details about each validation failure
        fmt.Printf("Validation failed: %s\n", err.Message)
        
        // Access detailed validation errors
        for field, details := range err.Details {
            fmt.Printf("Field %s: %v\n", field, details)
        }
        
        // Can also be used directly in HTTP responses
        // err.ToHTTP(responseWriter)
    }
}
```

### LLM Interaction

```go
package main

import (
    "context"
    "fmt"
    "io"
    "os"

    "github.com/Abraxas-365/craftable/ai/aiproviders"
    "github.com/Abraxas-365/craftable/ai/llm"
)

func main() {
    // Get API key from environment
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        fmt.Println("Please set OPENAI_API_KEY environment variable")
        os.Exit(1)
    }

    // Create the OpenAI provider
    provider := aiproviders.NewOpenAIProvider(apiKey)

    // Create a client with the provider
    client := llm.NewClient(provider)

    // Basic chat example
    messages := []llm.Message{
        llm.NewSystemMessage("You are a helpful assistant that provides concise answers."),
        llm.NewUserMessage("What's the capital of France?"),
    }

    resp, err := client.Chat(context.Background(), messages,
        llm.WithModel("gpt-4o"),
        llm.WithTemperature(0.7),
    )

    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // Print the response
    fmt.Println("Assistant:", resp.Message.Content)

    // Streaming example
    stream, err := client.ChatStream(context.Background(), messages,
        llm.WithModel("gpt-4o"),
        llm.WithTemperature(0.7),
    )

    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Print("Streaming response: ")
    for {
        msg, err := stream.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            fmt.Printf("\nStream error: %v\n", err)
            break
        }

        fmt.Print(msg.Content)
    }
    fmt.Println()
    stream.Close()
}
```

### Text Embedding

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/Abraxas-365/craftable/ai/aiproviders"
    "github.com/Abraxas-365/craftable/ai/embedding"
)

func main() {
    // Get API key from environment variable
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY environment variable not set")
    }

    // Create a new OpenAI provider
    provider := aiproviders.NewOpenAIProvider(apiKey)

    // Create embedding client
    embeddingClient := embedding.NewClient(provider)

    // Embed documents
    documents := []string{
        "The quick brown fox jumps over the lazy dog",
        "Paris is the capital of France",
        "Machine learning is a subset of artificial intelligence",
    }

    embeddings, err := embeddingClient.EmbedDocuments(
        context.Background(),
        documents,
        embedding.WithModel("text-embedding-3-small"),
        embedding.WithDimensions(1536),
    )
    if err != nil {
        log.Fatalf("Error generating embeddings: %v", err)
    }

    fmt.Printf("Generated %d embeddings\n", len(embeddings))
    fmt.Printf("First embedding dimensions: %d\n", len(embeddings[0].Vector))

    // Embed a query
    query := "What is artificial intelligence?"
    queryEmbedding, err := embeddingClient.EmbedQuery(
        context.Background(),
        query,
        embedding.WithModel("text-embedding-3-small"),
    )
    if err != nil {
        log.Fatalf("Error generating query embedding: %v", err)
    }

    fmt.Printf("Query embedding dimensions: %d\n", len(queryEmbedding.Vector))
    fmt.Printf("Usage: %+v\n", queryEmbedding.Usage)
}
```

### OCR Processing

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/Abraxas-365/craftable/ai/aiproviders"
    "github.com/Abraxas-365/craftable/ai/ocr"
)

func main() {
    // Get API key from environment variable
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY environment variable not set")
    }

    // Create a new OpenAI provider
    provider := aiproviders.NewOpenAIProvider(apiKey)

    // Create OCR client
    ocrClient := ocr.NewClient(provider)

    // Extract text from an image URL
    imageURL := "https://example.com/sample-image.jpg"
    
    result, err := ocrClient.ExtractTextFromURL(
        context.Background(),
        imageURL,
        ocr.WithModel("gpt-4-vision"),
        ocr.WithLanguage("auto"),
        ocr.WithDetailsLevel("high"),
    )
    if err != nil {
        log.Fatalf("Error extracting text from URL: %v", err)
    }

    fmt.Printf("Extracted Text:\n%s\n\n", result.Text)
    fmt.Printf("Overall Confidence: %.2f\n", result.Confidence)
    fmt.Printf("Number of Text Blocks: %d\n", len(result.Blocks))
    
    // Process text blocks
    for i, block := range result.Blocks {
        fmt.Printf("Block %d: '%s' (Confidence: %.2f)\n", 
            i+1, block.Text, block.Confidence)
    }
}
```

## 🎨 Design Principles

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
- [storex Documentation](https://pkg.go.dev/github.com/Abraxas-365/craftable/storex)
- [dtox Documentation](https://pkg.go.dev/github.com/Abraxas-365/craftable/dtox)
- [validatex Documentation](https://pkg.go.dev/github.com/Abraxas-365/craftable/validatex)
- [ai Documentation](https://pkg.go.dev/github.com/Abraxas-365/craftable/ai)

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
