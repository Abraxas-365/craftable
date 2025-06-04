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
    - [storex - Database Store Abstraction(Beta)](#storex---database-store-abstraction)
    - [dtox - DTO/Model Conversion](#dtox---dtomodel-conversion)
    - [validatex - Struct Validation](#validatex---struct-validation)
    - [docx - API Documentation Generator](#docx---api-documentation-generator)
    - [ai - Artificial Intelligence Toolkit](#ai---artificial-intelligence-toolkit)
      - [llm - Large Language Model Client](#llm---large-language-model-client)
        - [agentx - AI Agent Framework](#agentx---ai-agent-framework)
        - [guardrailx - Content Validation and Moderation](#guardrailx---content-validation-and-moderation)
        - [memoryx - Agent Memory Management](#memoryx---agent-memory-management)
        - [toolx - Agent Tool Framework](#toolx---agent-tool-framework)
      - [embedding - Text Embedding Interface](#embedding---text-embedding-interface)
      - [ocr - Optical Character Recognition](#ocr---optical-character-recognition)
      - [speech - Text-to-Speech and Speech-to-Text](#speech---text-to-speech-and-speech-to-text)
    - [configx- Configuration Management](#configx---advanced-configuration-management)

  - [🚀 Installation](#-installation)
  - [📝 Example Usage](#-example-usage)
    - [Error Handling (errx)](#error-handling-errx)
    - [Authentication (auth)](#authentication-auth)
    - [Database Store (storex)](#database-store-storex)
      - [Basic CRUD Operations](#basic-crud-operations)
      - [Pagination and Filtering](#pagination-and-filtering)
    - [DTO/Model Conversion (dtox)](#dtomodel-conversion-dtox)
    - [Struct Validation (validatex)](#struct-validation-validatex)
    - [API Documentation (docx)](#api-documentation-docx)
    - [LLM Interaction](#llm-interaction)
    - [Agent Interaction](#agent-interaction)
    - [Agent with Guardrails](#agent-with-guardrails)
    - [Text Embedding](#text-embedding)
    - [OCR Processing](#ocr-processing)
    - [Speech Processing](#speech-processing)
    - [Configuration Management(configx)](#configx-configuration-management)
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

### docx - API Documentation Generator

An intuitive API documentation generator that produces multiple formats from your code:

- **Code-First Approach**: Generate documentation directly from your API definitions
- **Multiple Output Formats**: Export to JSON, Markdown, HTML, and cURL examples
- **Fiber Integration**: Seamless integration with the Fiber web framework
- **Endpoint Schema Generation**: Automatically document request and response schemas
- **Authentication Documentation**: Document security requirements and token formats
- **Example Request/Response**: Include example payloads for better understanding
- **Struct Reflection**: Automatically extract DTO schemas from Go structs
- **Customizable Headers**: Document required and optional headers
- **Type-Safe Building**: Fluent, chainable API for documentation building

### ai - Artificial Intelligence Toolkit

A collection of packages for working with AI services and capabilities:

#### llm - Large Language Model Client

Interact with large language models with a clean, flexible interface:

- **Multiple Providers**: Support for various LLM providers (OpenAI, etc.)
- **Streaming Support**: Stream responses for real-time interactions
- **Message Management**: Structured conversation history handling
- **Tool Integration**: Support for function calling and tool usage
- **Type-safe Responses**: Strongly-typed message handling

##### agentx - AI Agent Framework

Build powerful AI agents with reasoning and tool-using capabilities:

- **Tool Integration**: Seamless integration with custom tools and functions
- **Memory Management**: Stateful conversations with system prompt management
- **Multi-turn Reasoning**: Handle complex, multi-step reasoning flows
- **Tool Execution Tracing**: Detailed execution traces for debugging
- **Interactive Mode**: Support for both batch and interactive conversations
- **Streaming Support**: Stream responses in real-time, even while using tools
- **Evaluation Framework**: Measure performance and analyze agent behaviors

##### guardrailx - Content Validation and Moderation

Apply rules and policies to control agent inputs and outputs:

- **Multiple Rule Types**: Use pattern matching, regex, LLM-based validation, and more
- **Bidirectional Protection**: Apply rules to both user inputs and agent responses
- **LLM-Based Validation**: Use AI to perform complex content moderation
- **Filtering Actions**: Block, modify, filter, or log policy violations
- **Customizable Responses**: Define appropriate fallback messages
- **Priority System**: Control rule evaluation order and importance

##### memoryx - Agent Memory Management

Flexible conversation memory for stateful agent interactions:

- **System Prompt Management**: Maintain and update system instructions
- **Context Window Management**: Automatically manage token limits
- **Conversation History**: Track the full history of interactions
- **Configurable Size**: Customize memory capacity based on requirements

##### toolx - Agent Tool Framework

Create and integrate tools that agents can use to perform actions:

- **Tool Interface**: Simple interface for defining agent tools
- **Type Conversion**: Automatic handling of different return types
- **Error Handling**: Structured error reporting from tool execution
- **Function Calling**: Compatible with LLM function calling capabilities
- **JSON Schema**: Automatic conversion of tools to JSON schema for LLMs

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

#### speech - Text-to-Speech and Speech-to-Text

Convert between text and speech with multiple voice and language options:

- **Text-to-Speech Synthesis**: Generate natural speech from text
- **Speech-to-Text Transcription**: Transcribe audio to text
- **Multiple Voices**: Support for different voices and speech styles
- **Speech Rate Control**: Adjust speed of generated speech
- **Audio Format Options**: Support for various audio formats (MP3, WAV, PCM)
- **Streaming Support**: Process audio in real-time
- **Provider Abstraction**: Consistent interface across speech providers

### configx - Advanced Configuration Management

A flexible configuration management system that supports multiple sources:

- **Multiple Sources**: Load configuration from environment variables, .env files, and more
- **Type Conversion**: Automatic conversion of string values to appropriate types
- **Nested Configuration**: Support for hierarchical configuration structures
- **Priority-Based Merging**: Higher priority sources override lower priority ones
- **Unified Access API**: Consistent access to configuration values regardless of source
- **Environment Variable Support**: Load configuration from environment variables with prefix filtering
- **Validation**: Verify required configuration values are present
- **Change Notification**: React to configuration changes
- **Auto-reload**: Automatically reload configuration at intervals

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
    "github.com/Abraxas-365/craftable/auth/providers/authgoogle"
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
    googleProvider := authgoogle.NewGoogleProvider(
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
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// User represents a user in our application
type User struct {
	ID        string    `db:"id" bson:"_id"`
	Email     string    `db:"email" bson:"email"`
	Name      string    `db:"name" bson:"name"`
	Age       int       `db:"age" bson:"age"`
	Active    bool      `db:"active" bson:"active"`
	CreatedAt time.Time `db:"created_at" bson:"created_at"`
}

func main() {
	ctx := context.Background()

	// Choose one of these initialization blocks based on your database:
	
	// PostgreSQL initialization
	db, err := sqlx.Connect("postgres", "postgres://username:password@localhost/mydatabase?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	userRepo := storexpostgres.NewPgRepository[User](db, "users", "id")

	// MongoDB initialization
	/*
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	collection := storexmongomongoClient.Database("mydatabase").Collection("users")
	userRepo := storex.NewMongoRepository[User](collection, "_id")
	*/

	// Create a new user
	newUser := User{
		Email:     "john@example.com",
		Name:      "John Doe",
		Age:       30,
		Active:    true,
		CreatedAt: time.Now(),
	}

	createdUser, err := userRepo.Create(ctx, newUser)
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}
	fmt.Printf("Created user with ID: %s\n", createdUser.ID)

	// Find user by ID
	user, err := userRepo.FindByID(ctx, createdUser.ID)
	if err != nil {
		log.Fatalf("Failed to find user: %v", err)
	}
	fmt.Printf("Found user: %s (%s)\n", user.Name, user.Email)

	// Find one with filter
	filterUser, err := userRepo.FindOne(ctx, map[string]any{
		"email": "john@example.com",
	})
	if err != nil {
		log.Fatalf("Failed to find user by email: %v", err)
	}
	fmt.Printf("Found user by email: %s\n", filterUser.Name)

	// Update user
	user.Name = "John Smith"
	updatedUser, err := userRepo.Update(ctx, user.ID, user)
	if err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}
	fmt.Printf("Updated user name to: %s\n", updatedUser.Name)

	// Delete user
	err = userRepo.Delete(ctx, user.ID)
	if err != nil {
		log.Fatalf("Failed to delete user: %v", err)
	}
	fmt.Println("User deleted successfully")
}
```

#### Pagination and Filtering

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx := context.Background()

	// Initialize your chosen database (see previous example)
	
	// For this example, we'll assume PostgreSQL
	db, err := sqlx.Connect("postgres", "postgres://username:password@localhost/mydatabase?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	userRepo := storex.NewPgRepository[User](db, "users", "id")

	// Basic pagination - get first page of users
	paginationOpts := storex.PaginationOptions{
		Page:     1,
		PageSize: 10,
		OrderBy:  "created_at",
		Desc:     true, // newest first
	}

	result, err := userRepo.Paginate(ctx, paginationOpts)
	if err != nil {
		log.Fatalf("Pagination failed: %v", err)
	}

	fmt.Printf("Found %d users (page %d of %d)\n", 
		result.Page.Total, result.Page.Number, result.Page.Pages)
	
	for i, user := range result.Data {
		fmt.Printf("%d. %s (%s)\n", i+1, user.Name, user.Email)
	}

	// Pagination with filtering - active users over 25
	paginationWithFilters := storex.DefaultPaginationOptions().
		WithFilter("active", true).
		WithFilter("age", map[string]any{"$gt": 25}) // MongoDB style filter (works with PG adapter too)

	filteredResult, err := userRepo.Paginate(ctx, paginationWithFilters)
	if err != nil {
		log.Fatalf("Filtered pagination failed: %v", err)
	}

	fmt.Printf("Found %d active users over 25\n", filteredResult.Page.Total)
	
	// Using query builder for more complex pagination
	qb := storex.NewQueryBuilder[User]().
		Where("active", "=", true).
		Where("age", ">", 25).
		OrderBy("name", false). // ascending
		Limit(10).
		Select("id", "name", "email")
	
	builderResult, err := userRepo.Paginate(ctx, qb.ToPaginationOptions())
	if err != nil {
		log.Fatalf("Query builder pagination failed: %v", err)
	}
	
	fmt.Printf("Found %d users with query builder\n", builderResult.Page.Total)
	
	// Navigation through pages
	if builderResult.HasNext() {
		nextPageOpts := qb.ToPaginationOptions()
		nextPageOpts.Page++
		
		nextPage, err := userRepo.Paginate(ctx, nextPageOpts)
		if err != nil {
			log.Fatalf("Next page retrieval failed: %v", err)
		}
		
		fmt.Printf("Next page has %d users\n", len(nextPage.Data))
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
### API Documentation (docx)

```go
import "github.com/Abraxas-365/craftable/docx"

// Define your DTOs
type UserCreateRequest struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

type UserResponse struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Create a router documentation
apiDoc := docx.NewRouterDoc("/api/v1")

// Document an endpoint with authentication
createUserEndpoint := docx.NewEndpoint("/users", docx.POST).
    WithDescription("Create a new user").
    WithSummary("User Registration").
    WithTags("users", "auth").
    WithRequestDTO(UserCreateRequest{}).
    WithResponseDTO(UserResponse{}).
    WithAuth(docx.Bearer, map[string]string{
        "description": "JWT token required",
        "scope": "admin",
    }).
    WithHeader("Content-Type", "application/json", true).
    WithRequestExample(UserCreateRequest{
        Name:     "John Doe",
        Email:    "john@example.com",
        Password: "secure123",
    }).
    WithResponseExample(UserResponse{
        ID:    "123",
        Name:  "John Doe",
        Email: "john@example.com",
    })

// Add endpoint to router doc
apiDoc.AddEndpoint(createUserEndpoint)

// Create a generator
generator := docx.NewGenerator().AddRouter(apiDoc)

// Generate JSON documentation
generator.GenerateJSON("docs/api.json")

// Generate curl documentation
generator.GenerateCurlDocs("http://localhost:3000", "docs/curl.md")

// Register documentation in Fiber app
apiDoc.RegisterWithFiber(app, "/api/docs")
```


### LLM Interaction

```go
package main

import (
    "context"
    "fmt"
    "io"
    "os"

    "github.com/Abraxas-365/craftable/ai/providers/aiopenai"
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
    provider := aiopenai.NewOpenAIProvider(apiKey)

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

### Agent Interaction

```go
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Abraxas-365/craftable/ai/providers/aiopenai"
	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/llm/agentx"
	"github.com/Abraxas-365/craftable/ai/llm/memoryx"
	"github.com/Abraxas-365/craftable/ai/llm/toolx"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY environment variable")
		os.Exit(1)
	}

	// Create the LLM client using your existing provider
	provider := aiopenai.NewOpenAIProvider(apiKey)
	client := llm.NewClient(provider)

	// Create a simple tool
	weatherTool := NewWeatherTool()
	tools := toolx.FromToolx(weatherTool)

	// Create memory with system prompt
	mem := memoryx.NewMemory(memoryx.WithSystemPrompt("You are a helpful assistant that can check weather conditions."))
	// Create the agent
	myAgent := agentx.New(
		client,
		mem,
		agentx.WithTools(tools),
		agentx.WithOptions(
			// llm.WithModel("gpt-4o"),
			llm.WithMaxTokens(500),
			llm.WithTemperature(0.7),
		),
	)

	fmt.Println("=== Interactive Weather Assistant ===")
	fmt.Println("Type your questions about weather (press Ctrl+C to exit)")
	fmt.Println("Example: What's the weather like in New York?")

	// Create a scanner to read user input
	scanner := bufio.NewScanner(os.Stdin)

	// Continuous input loop
	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break // Exit on EOF
		}

		userQuery := scanner.Text()
		if strings.TrimSpace(userQuery) == "" {
			continue // Skip empty queries
		}

		// Process the query
		response, err := myAgent.EvaluateWithTools(context.Background(), userQuery)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Print just the final response
		fmt.Println("\nAssistant:", response.FinalResponse)

		// Optional: uncomment this block if you want to see the detailed execution trace
		/*
			eval, err := myAgent.EvaluateWithTools(context.Background(), userQuery)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			fmt.Println("\n--- Execution Steps ---")
			for i, step := range eval.Steps {
				fmt.Printf("\nStep %d (%s):\n", i+1, step.StepType)

				if step.StepType == "initial" || step.StepType == "response" {
					fmt.Printf("LLM Output: %s\n", step.OutputMessage.Content)

					if len(step.OutputMessage.ToolCalls) > 0 {
						fmt.Println("Tool Calls:")
						for _, tc := range step.OutputMessage.ToolCalls {
							fmt.Printf("  - %s: %s\n", tc.Function.Name, tc.Function.Arguments)
						}
					}

					fmt.Printf("Tokens Used: %d\n", step.TokenUsage.TotalTokens)
				}

				if step.StepType == "tool_execution" {
					fmt.Println("Tool Responses:")
					for _, tr := range step.ToolResponses {
						fmt.Printf("  - %s\n", tr.Content)
					}
				}
			}
		*/
	}

	fmt.Println("\nExiting. Goodbye!")
}

// WeatherTool provides weather information
type WeatherTool struct{}

type WeatherRequest struct {
	Location string `json:"location"`
}

func NewWeatherTool() *WeatherTool {
	return &WeatherTool{}
}

func (w *WeatherTool) Name() string {
	return "get_weather"
}

func (w *WeatherTool) GetTool() llm.Tool {
	return llm.Tool{
		Type: "function",
		Function: llm.Function{
			Name:        w.Name(),
			Description: "Get the current weather in a location",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city name, e.g. New York",
					},
				},
				"required": []string{"location"},
			},
		},
	}
}

func (w *WeatherTool) Call(ctx context.Context, inputs string) (any, error) {
	// Parse input
	var request WeatherRequest
	if err := json.Unmarshal([]byte(inputs), &request); err != nil {
		return nil, fmt.Errorf("failed to parse weather request: %w", err)
	}

	// Simple mock implementation - in a real app, you'd call a weather API
	weatherData := fmt.Sprintf("Currently 22Â°C and partly cloudy in %s.", request.Location)
	return weatherData, nil
}
```

### Agent with Guardrails

```go
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Abraxas-365/craftable/ai/providers/aiopenai"
	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/llm/agentx"
	"github.com/Abraxas-365/craftable/ai/llm/guardrailx"
	"github.com/Abraxas-365/craftable/ai/llm/memoryx"
	"github.com/Abraxas-365/craftable/ai/llm/toolx"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY environment variable")
		os.Exit(1)
	}

	// Create the LLM client
	provider := aiopenai.NewOpenAIProvider(apiKey)
	llmClient := llm.NewClient(provider)

	// Create tools
	weatherTool := NewWeatherTool()
	tools := toolx.FromToolx(weatherTool)

	// Create memory
	mem := memoryx.NewMemory(memoryx.WithSystemPrompt("You are a helpful assistant that can check weather conditions."))

	// Create the agent
	myAgent := agentx.New(
		llmClient,
		mem,
		agentx.WithTools(tools),
		agentx.WithOptions(
			llm.WithModel("gpt-4o"),
			llm.WithMaxTokens(500),
		),
	)

	// Define guardrail rules
	rules := []guardrailx.Rule{
		{
			Name:        "Profanity Filter",
			Type:        guardrailx.BlocklistRule,
			Pattern:     []string{"damn", "hell", "shit", "fuck"},
			Direction:   guardrailx.Both,
			Action:      guardrailx.FilterAction,
			Replacement: "****",
			Message:     "Please avoid using profanity.",
			Priority:    90,
		},
		// LLM Rule to check if input is weather-related
		{
			Name:      "Weather Topic Classifier",
			Type:      guardrailx.LLMRule,
			Direction: guardrailx.Input,
			Action:    guardrailx.BlockAction,
			Message:   "I'm a weather assistant. Please ask me questions about weather, climate, or meteorological conditions.",
			Priority:  100, // Higher priority than the keyword check
			LLMConfig: &guardrailx.LLMCheckConfig{
				Client: llmClient,
				InputPrompt: `Determine if the following user query is related to weather, climate, or meteorological information:

{{content}}

A query is weather-related if it asks about:
- Current weather conditions in any location
- Weather forecasts or predictions
- Temperature, precipitation, humidity, wind, air pressure
- Climate patterns or historical weather data
- Meteorological phenomena (storms, hurricanes, etc.)
- Any conditions related to the atmosphere

Examples of weather-related queries:
- "What's the weather in Paris?"
- "Will it rain tomorrow in Seattle?"
- "Is it hot in Arizona right now?"
- "What's the forecast for this weekend?"
- "How's the climate in Peru?"
- "Tell me about hurricane formation"

Respond with ONLY "yes" if the query is related to weather or "no" if it's about something else.`,
				ModelOptions: []llm.Option{
					llm.WithModel("gpt-4o-mini"), // Use smaller model for guardrails to save costs
					llm.WithTemperature(0.0),     // Low temperature for consistent moderation
					llm.WithMaxTokens(10),        // Only need a short response
				},
				ValidResponse: []string{"yes"}, // Only "yes" is considered valid
			},
		},
	}

	// Create guardrail
	guardrail := guardrailx.NewGuardRail(
		rules,
		guardrailx.WithLogger(&CustomLogger{}),
		guardrailx.WithDefaultLLM(llmClient, []llm.Option{
			llm.WithModel("gpt-3.5-turbo"),
			llm.WithTemperature(0.0),
		}),
	)

	// Create guarded agent
	guardedAgent := guardrailx.NewGuardedAgent(myAgent, guardrail, nil)

	// Interactive chat loop
	fmt.Println("=== Weather Assistant with LLM Guardrails ===")
	fmt.Println("Type your questions (press Ctrl+C to exit)")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		if strings.TrimSpace(input) == "" {
			continue
		}

		response, err := guardedAgent.Run(context.Background(), input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Println("\nA:", response)
	}
}

// CustomLogger logs guardrail violations
type CustomLogger struct{}

func (l *CustomLogger) Log(rule guardrailx.Rule, content string, direction guardrailx.Direction, result guardrailx.ValidationResult) error {
	fmt.Printf("[GUARDRAIL] %s rule '%s' triggered for %s message\n", rule.Type, rule.Name, direction)
	if result.Details != "" {
		fmt.Printf("  LLM response: %s\n", result.Details)
	}
	return nil
}

type WeatherTool struct{}

type WeatherRequest struct {
	Location string `json:"location"`
}

func NewWeatherTool() *WeatherTool {
	return &WeatherTool{}
}

func (w *WeatherTool) Name() string {
	return "get_weather"
}

func (w *WeatherTool) GetTool() llm.Tool {
	return llm.Tool{
		Type: "function",
		Function: llm.Function{
			Name:        w.Name(),
			Description: "Get the current weather in a location",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city name, e.g. New York",
					},
				},
				"required": []string{"location"},
			},
		},
	}
}

func (w *WeatherTool) Call(ctx context.Context, inputs string) (any, error) {
	// Parse input
	var request WeatherRequest
	if err := json.Unmarshal([]byte(inputs), &request); err != nil {
		return nil, fmt.Errorf("failed to parse weather request: %w", err)
	}

	// Fixed temperature symbol
	weatherData := fmt.Sprintf("Currently 22°C and partly cloudy in %s.", request.Location)
	return weatherData, nil
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

    "github.com/Abraxas-365/craftable/ai/providers/aiopenai"
    "github.com/Abraxas-365/craftable/ai/embedding"
)

func main() {
    // Get API key from environment variable
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY environment variable not set")
    }

    // Create a new OpenAI provider
    provider := aiopenai.NewOpenAIProvider(apiKey)

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

    "github.com/Abraxas-365/craftable/ai/providers/aiopenai"
    "github.com/Abraxas-365/craftable/ai/ocr"
)

func main() {
    // Get API key from environment variable
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY environment variable not set")
    }

    // Create a new OpenAI provider
    provider := aiopenai.NewOpenAIProvider(apiKey)

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
### Speech Processing

```go
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Abraxas-365/craftable/ai/providers/aiopenai"
	"github.com/Abraxas-365/craftable/ai/speech"
	"github.com/openai/openai-go"
)

func main() {
	// Get API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	
	// Create provider and clients
	provider := aiopenai.NewOpenAIProvider(apiKey)
	ttsClient := speech.NewTTSClient(provider)
	sttClient := speech.NewSTTClient(provider)
	
	// Text-to-Speech example
	text := "Hello world! This is text converted to speech."
	audio, err := ttsClient.Synthesize(context.Background(), text,
		speech.WithTTSModel(string(openai.SpeechModelTTS1HD)),
		speech.WithVoice("nova"),
		speech.WithOutputFormat(speech.AudioFormatMP3),
	)
	if err != nil {
		fmt.Printf("Synthesis error: %v\n", err)
		return
	}
	defer audio.Content.Close()
	
	// Save to file
	outputFile := "output.mp3"
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()
	
	if _, err := io.Copy(file, audio.Content); err != nil {
		fmt.Printf("Error saving audio: %v\n", err)
		return
	}
	
	fmt.Printf("Audio saved to %s\n", outputFile)
	fmt.Printf("Sample rate: %d Hz, Format: %s\n", audio.SampleRate, audio.Format)
	
	// Speech-to-Text example
	inputFile := "input.mp3"
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		// If the input file doesn't exist, use the file we just created
		fmt.Printf("Input file not found, using generated audio: %s\n", outputFile)
		inputFile = outputFile
	}
	
	transcript, err := sttClient.TranscribeFile(context.Background(), inputFile,
		speech.WithSTTModel(string(openai.AudioModelWhisper1)),
		speech.WithLanguage("en"),
	)
	if err != nil {
		fmt.Printf("Transcription error: %v\n", err)
		return
	}
	
	fmt.Println("Transcribed text:", transcript.Text)
	
	// Example of using the Transcribe method with a file reader
	file, err = os.Open(inputFile)
	if err != nil {
		fmt.Printf("Error opening audio file: %v\n", err)
		return
	}
	defer file.Close()
	
	transcript, err = sttClient.Transcribe(context.Background(), file,
		speech.WithSTTModel(string(openai.AudioModelWhisper1)),
	)
	if err != nil {
		fmt.Printf("Direct transcription error: %v\n", err)
		return
	}
	
	fmt.Println("Direct transcription result:", transcript.Text)
}
```

### configx - Configuration Management

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Abraxas-365/craftable/configx"
)

func main() {
	// Set environment variables programmatically
	os.Setenv("APP_SERVER_PORT", "9000")
	os.Setenv("APP_SERVER_HOST", "api.example.com")
	os.Setenv("APP_DATABASE_URL", "postgres://user:pass@db.example.com:5432/mydb")
	os.Setenv("APP_API_KEY", "my-secret-api-key")
	os.Setenv("APP_DEBUG", "true")
	os.Setenv("APP_MAX_CONNECTIONS", "100")

	// Create configuration from environment variables and defaults
	config, err := configx.NewBuilder().
		WithDefaults(map[string]interface{}{
			"server": map[string]interface{}{
				"port": 8080,
				"host": "localhost",
			},
			"debug": false,
		}).
		FromEnv("APP_").
		RequireEnv("APP_DATABASE_URL", "APP_API_KEY").
		Build()

	if err != nil {
		log.Fatalf("Configuration error: %s", err)
	}

	// Access configuration values with type conversion
	serverPort := config.Get("server.port").AsInt()
	serverHost := config.Get("server.host").AsString()
	dbURL := config.Get("database.url").AsString()
	apiKey := config.Get("api.key").AsString()
	debug := config.Get("debug").AsBool()

	fmt.Printf("Server:       %s:%d\n", serverHost, serverPort)
	fmt.Printf("Database URL: %s\n", dbURL)
	fmt.Printf("API Key:      %s\n", apiKey)
	fmt.Printf("Debug Mode:   %t\n", debug)
}
```

#### Loading from Multiple Sources
```go
package main

import (
	"fmt"
	"log"

	"github.com/Abraxas-365/craftable/configx"
)

func main() {
	// Create configuration with multiple sources
	config, err := configx.NewBuilder().
		WithDefaults(map[string]interface{}{
			"server": map[string]interface{}{
				"port": 8080,
			},
			"timeout": 30,
		}).
		FromDotEnv(".env").          // Load from .env file
		FromEnv("APP_").             // Load from environment variables
		FromFile("config.yaml").     // Load from YAML file
		Build()

	if err != nil {
		log.Fatalf("Configuration error: %s", err)
	}

	// Access configuration with default values
	port := config.Get("server.port").AsIntDefault(9000)
	timeout := config.Get("timeout").AsIntDefault(60)
	
	fmt.Printf("Server Port: %d\n", port)
	fmt.Printf("Timeout: %d seconds\n", timeout)
	
	// Check if a configuration value exists
	if config.Has("database.url") {
		fmt.Printf("Database URL: %s\n", config.Get("database.url").AsString())
	} else {
		fmt.Println("No database URL configured")
	}
}
```

### Type Conversion and Nested Configuration
```go
package main

import (
	"fmt"
	"time"

	"github.com/Abraxas-365/craftable/configx"
)

func main() {
	// Sample configuration with different types
	cfg, _ := configx.NewBuilder().
		WithDefaults(map[string]interface{}{
			"server": map[string]interface{}{
				"timeouts": map[string]interface{}{
					"read":    "5s",
					"write":   "10s",
					"idle":    "60s",
				},
				"maxConnections": 100,
				"tls": map[string]interface{}{
					"enabled": true,
					"certs": []string{
						"/path/to/cert1",
						"/path/to/cert2",
					},
				},
			},
		}).
		Build()

	// Access values with type conversion
	readTimeout := cfg.Get("server.timeouts.read").AsDuration()
	writeTimeout := cfg.Get("server.timeouts.write").AsDuration()
	maxConn := cfg.Get("server.maxConnections").AsInt()
	tlsEnabled := cfg.Get("server.tls.enabled").AsBool()
	
	fmt.Printf("Read Timeout: %s\n", readTimeout)
	fmt.Printf("Write Timeout: %s\n", writeTimeout)
	fmt.Printf("Max Connections: %d\n", maxConn)
	fmt.Printf("TLS Enabled: %t\n", tlsEnabled)
	
	// Access array values
	certs := cfg.Get("server.tls.certs").AsStringSlice()
	for i, cert := range certs {
		fmt.Printf("Certificate %d: %s\n", i+1, cert)
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
