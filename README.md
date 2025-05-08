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

## 📥 Installation

```bash
go get github.com/Abraxas-365/craftable
```

Or install specific packages:

```bash
go get github.com/Abraxas-365/craftable/errx
go get github.com/Abraxas-365/craftable/auth
go get github.com/Abraxas-365/craftable/storex
```

## 🚀 Example Usage

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

#### Advanced Features: Bulk Operations, Transactions, Search

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    
    "github.com/Abraxas-365/craftable/storex"
    _ "github.com/lib/pq" // PostgreSQL driver
)

// Define product model
type Product struct {
    ID          int     `json:"id"`
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Price       float64 `json:"price"`
    CreatedAt   string  `json:"created_at"`
}

func ExampleAdvancedFeatures(db *sql.DB) {
    // Create a typed store for Products
    productStore := storex.NewTypedSQL[Product](db).
        WithTableName("products").
        WithIDColumn("id")

    ctx := context.Background()

    // 1. Bulk operations
    products := []Product{
        {Name: "Product 1", Price: 19.99},
        {Name: "Product 2", Price: 29.99},
        {Name: "Product 3", Price: 39.99},
    }
    
    // Insert multiple products at once
    err := productStore.BulkInsert(ctx, products)
    if err != nil {
        // Handle error
        return
    }
    
    // 2. Transaction support
    err = productStore.WithTransaction(ctx, func(txCtx context.Context) error {
        // All operations in this function are part of the same transaction
        product := Product{Name: "Transaction Product", Price: 99.99}
        
        created, err := productStore.Create(txCtx, product)
        if err != nil {
            return err // Will cause rollback
        }
        
        idStr := fmt.Sprintf("%d", created.ID)
        return productStore.Delete(txCtx, idStr) // Success = commit, error = rollback
    })
    
    // 3. Full-text search
    searchOpts := storex.SearchOptions{
        Fields: []string{"name", "description"},
        Limit:  10,
    }
    
    results, err := productStore.Search(ctx, "smartphone", searchOpts)
    if err != nil {
        // Handle error
        return
    }
    
    fmt.Printf("Found %d matching products\n", len(results))
    
    // 4. Query building
    query := storex.NewQueryBuilder[Product]().
        Where("price", ">", 50.0).
        Where("name", "LIKE", "%phone%").
        OrderBy("price", false).  // ascending order
        Limit(20)
        
    // Convert to pagination options
    queryOpts := query.ToPaginationOptions()
    queryResults, err := productStore.Paginate(ctx, queryOpts)
    
    if err != nil {
        // Handle error
        return
    }
    
    fmt.Printf("Found %d matching products\n", queryResults.Page.Total)
}
```

## 🔍 Design Principles

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

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
