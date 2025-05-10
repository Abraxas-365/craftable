// Package storex provides a generic abstraction layer for working with different database stores.
//
// The package currently supports MongoDB and SQL databases, with a focus on providing
// a consistent interface for database operations across these database types.
// It includes robust error handling, type-safe generic implementations, and utility
// functions to simplify common database operations.
//
// Key Features:
//   - Type-safe generic implementations for database operations
//   - Complete CRUD operations with a unified API
//   - Powerful pagination with sorting and filtering
//   - Bulk operations for improved performance
//   - Transaction support
//   - Full-text search capabilities
//   - Query builder for type-safe queries
//   - Real-time data change notifications
//   - Consistent error handling with detailed context
//
// Basic CRUD Operations:
//
//	import (
//		"context"
//		"github.com/Abraxas-365/craftable/storex"
//		"go.mongodb.org/mongo-driver/mongo"
//	)
//
//	// Define your model
//	type User struct {
//		ID        string `bson:"_id,omitempty"`
//		Name      string `bson:"name"`
//		Email     string `bson:"email"`
//		CreatedAt int64  `bson:"created_at"`
//	}
//
//	func ExampleCRUDOperations(client *mongo.Client) {
//		// Get collection
//		collection := client.Database("myapp").Collection("users")
//
//		// Create a typed store for Users
//		userStore := storex.NewTypedMongo[User](collection)
//
//		ctx := context.Background()
//
//		// Create a new user
//		newUser := User{
//			Name:      "John Doe",
//			Email:     "john@example.com",
//			CreatedAt: time.Now().Unix(),
//		}
//
//		createdUser, err := userStore.Create(ctx, newUser)
//		if err != nil {
//			// Handle error
//			return
//		}
//
//		// Find user by ID
//		user, err := userStore.FindByID(ctx, createdUser.ID)
//		if err != nil {
//			if storex.IsRecordNotFound(err) {
//				// Handle not found case
//				return
//			}
//			// Handle other errors
//			return
//		}
//
//		// Update user
//		user.Name = "John Smith"
//		updatedUser, err := userStore.Update(ctx, user.ID, user)
//		if err != nil {
//			// Handle error
//			return
//		}
//
//		// Find one user by filter
//		filter := map[string]any{"email": "john@example.com"}
//		foundUser, err := userStore.FindOne(ctx, filter)
//		if err != nil {
//			// Handle error
//			return
//		}
//
//		// Delete user
//		err = userStore.Delete(ctx, user.ID)
//		if err != nil {
//			// Handle error
//			return
//		}
//	}
//
// MongoDB Pagination:
//
//	import (
//		"context"
//		"fmt"
//		"github.com/Abraxas-365/craftable/storex"
//		"go.mongodb.org/mongo-driver/mongo"
//	)
//
//	func ExampleMongoDBPagination(client *mongo.Client) {
//		// Get collection
//		collection := client.Database("myapp").Collection("users")
//
//		// Create a typed store for Users
//		userStore := storex.NewTypedMongo[User](collection)
//
//		// Create pagination options
//		opts := storex.DefaultPaginationOptions()
//		opts.Page = 1
//		opts.PageSize = 10
//		opts.OrderBy = "created_at"
//		opts.Desc = true
//
//		// Add filters
//		opts = opts.WithFilter("name", "John")
//
//		// Perform paginated query
//		ctx := context.Background()
//		result, err := userStore.Paginate(ctx, opts)
//		if err != nil {
//			// Handle error (check if record not found)
//			if storex.IsRecordNotFound(err) {
//				// Handle not found case
//				return
//			}
//			// Handle other errors
//			return
//		}
//
//		// Access results
//		for _, user := range result.Data {
//			// Process each user
//			fmt.Println(user.Name, user.Email)
//		}
//
//		// Use pagination metadata
//		fmt.Printf("Page %d of %d (Total: %d items)\n",
//			result.Page.Number, result.Page.Pages, result.Page.Total)
//
//		// Check if there are more pages
//		if result.HasNext() {
//			// Fetch next page
//		}
//	}
//
// SQL Operations:
//
//	import (
//		"context"
//		"database/sql"
//		"fmt"
//		"github.com/Abraxas-365/craftable/storex"
//		_ "github.com/lib/pq" // PostgreSQL driver
//	)
//
//	// Define your model
//	type Product struct {
//		ID          int     `json:"id"`
//		Name        string  `json:"name"`
//		Description string  `json:"description"`
//		Price       float64 `json:"price"`
//		CreatedAt   string  `json:"created_at"`
//	}
//
//	func ExampleSQLOperations(db *sql.DB) {
//		// Create a typed store for Products
//		productStore := storex.NewTypedSQL[Product](db).
//			WithTableName("products").
//			WithIDColumn("id")
//
//		ctx := context.Background()
//
//		// Create a new product
//		newProduct := Product{
//			Name:        "Smartphone",
//			Description: "Latest model",
//			Price:       999.99,
//			CreatedAt:   time.Now().Format(time.RFC3339),
//		}
//
//		createdProduct, err := productStore.Create(ctx, newProduct)
//		if err != nil {
//			// Handle error
//			return
//		}
//
//		// Find product by ID
//		productID := fmt.Sprintf("%d", createdProduct.ID)
//		product, err := productStore.FindByID(ctx, productID)
//		if err != nil {
//			// Handle error
//			return
//		}
//
//		// Pagination
//		opts := storex.DefaultPaginationOptions()
//		opts.Page = 1
//		opts.PageSize = 20
//		opts.OrderBy = "price"
//		opts.Desc = true
//
//		// Add filters
//		opts = opts.WithFilter("price", 500.0)
//
//		result, err := productStore.Paginate(ctx, opts)
//		if err != nil {
//			// Handle error
//			return
//		}
//
//		fmt.Printf("Found %d expensive products\n", result.Page.Total)
//
//		// Simple pagination (using reflection-based scanning)
//		query := "SELECT id, name, description, price, created_at FROM products WHERE price > $1"
//		args := []interface{}{10.0} // Products with price > 10.0
//
//		simpleResult, err := productStore.PaginateSimple(ctx, opts, query, args)
//		if err != nil {
//			// Handle error
//			return
//		}
//
//		// Display pagination info
//		fmt.Printf("Showing %d of %d products (Page %d/%d)\n",
//			len(simpleResult.Data), simpleResult.Page.Total,
//			simpleResult.Page.Number, simpleResult.Page.Pages)
//	}
//
// Bulk Operations:
//
//	import (
//		"context"
//		"github.com/Abraxas-365/craftable/storex"
//		"go.mongodb.org/mongo-driver/mongo"
//	)
//
//	func ExampleBulkOperations(client *mongo.Client) {
//		// Create a typed store
//		collection := client.Database("myapp").Collection("users")
//		userStore := storex.NewTypedMongo[User](collection)
//
//		ctx := context.Background()
//
//		// Prepare data
//		users := []User{
//			{Name: "User 1", Email: "user1@example.com", CreatedAt: time.Now().Unix()},
//			{Name: "User 2", Email: "user2@example.com", CreatedAt: time.Now().Unix()},
//			{Name: "User 3", Email: "user3@example.com", CreatedAt: time.Now().Unix()},
//		}
//
//		// Bulk insert
//		err := userStore.BulkInsert(ctx, users)
//		if err != nil {
//			// Handle error
//			return
//		}
//
//		// Fetch users for bulk update
//		opts := storex.DefaultPaginationOptions()
//		opts.PageSize = 100 // Get a batch of users
//		result, err := userStore.Paginate(ctx, opts)
//		if err != nil {
//			// Handle error
//			return
//		}
//
//		// Modify users
//		for i := range result.Data {
//			result.Data[i].Name = "Updated " + result.Data[i].Name
//		}
//
//		// Bulk update
//		err = userStore.BulkUpdate(ctx, result.Data)
//		if err != nil {
//			// Handle error
//			return
//		}
//
//		// Collect IDs for bulk delete
//		ids := make([]string, len(result.Data))
//		for i, user := range result.Data {
//			ids[i] = user.ID
//		}
//
//		// Bulk delete
//		err = userStore.BulkDelete(ctx, ids)
//		if err != nil {
//			// Handle error
//			return
//		}
//	}
//
// Transaction Support:
//
//	import (
//		"context"
//		"database/sql"
//		"github.com/Abraxas-365/craftable/storex"
//	)
//
//	func ExampleTransactions(db *sql.DB) {
//		productStore := storex.NewTypedSQL[Product](db).
//			WithTableName("products")
//
//		ctx := context.Background()
//
//		// Execute operations in a transaction
//		err := productStore.WithTransaction(ctx, func(txCtx context.Context) error {
//			// Create a product
//			product := Product{
//				Name:  "Transaction Test",
//				Price: 99.99,
//			}
//
//			created, err := productStore.Create(txCtx, product)
//			if err != nil {
//				return err // Transaction will be rolled back
//			}
//
//			// Update the product
//			created.Price = 199.99
//			_, err = productStore.Update(txCtx, fmt.Sprintf("%d", created.ID), created)
//			if err != nil {
//				return err // Transaction will be rolled back
//			}
//
//			return nil // Transaction will be committed
//		})
//
//		if err != nil {
//			// Handle transaction error
//			return
//		}
//	}
//
// Query Builder:
//
//	import (
//		"context"
//		"github.com/Abraxas-365/craftable/storex"
//		"go.mongodb.org/mongo-driver/mongo"
//	)
//
//	func ExampleQueryBuilder(client *mongo.Client) {
//		collection := client.Database("myapp").Collection("users")
//		userStore := storex.NewTypedMongo[User](collection)
//
//		ctx := context.Background()
//
//		// Build a complex query
//		query := storex.NewQueryBuilder[User]().
//			Where("age", ">", 25).
//			Where("status", "=", "active").
//			OrderBy("created_at", true). // true = descending
//			Limit(50).
//			Select("name", "email", "created_at")
//
//		// Convert to pagination options
//		opts := query.ToPaginationOptions()
//
//		// Execute query
//		result, err := userStore.Paginate(ctx, opts)
//		if err != nil {
//			// Handle error
//			return
//		}
//
//		fmt.Printf("Found %d users matching criteria\n", result.Page.Total)
//	}
//
// Search Functionality:
//
//	import (
//		"context"
//		"fmt"
//		"github.com/Abraxas-365/craftable/storex"
//	)
//
//	func ExampleSearch(db *sql.DB) {
//		productStore := storex.NewTypedSQL[Product](db).
//			WithTableName("products")
//
//		ctx := context.Background()
//
//		// Configure search options
//		searchOpts := storex.SearchOptions{
//			Fields: []string{"name", "description"},
//			Limit:  20,
//			Boost: map[string]float64{
//				"name": 2.0, // Boost matches in name field
//			},
//		}
//
//		// Perform search
//		results, err := productStore.Search(ctx, "smartphone", searchOpts)
//		if err != nil {
//			// Handle error
//			return
//		}
//
//		fmt.Printf("Found %d matching products\n", len(results))
//		for _, product := range results {
//			fmt.Println(product.Name, product.Price)
//		}
//	}
//
// Change Streams (MongoDB):
//
//	import (
//		"context"
//		"fmt"
//		"github.com/Abraxas-365/craftable/storex"
//		"go.mongodb.org/mongo-driver/mongo"
//	)
//
//	func ExampleChangeStream(client *mongo.Client) {
//		collection := client.Database("myapp").Collection("users")
//		userStore := storex.NewTypedMongo[User](collection)
//
//		ctx, cancel := context.WithCancel(context.Background())
//		defer cancel()
//
//		// Watch for changes to users with email from a specific domain
//		filter := map[string]any{
//			"fullDocument.email": map[string]any{
//				"$regex": "@example.com$",
//			},
//		}
//
//		// Start watching for changes
//		changeStream, err := userStore.Watch(ctx, filter)
//		if err != nil {
//			// Handle error
//			return
//		}
//
//		// Process change events in a goroutine
//		go func() {
//			for event := range changeStream {
//				switch event.Operation {
//				case "insert":
//					fmt.Printf("New user created: %s\n", event.NewValue.Name)
//				case "update":
//					fmt.Printf("User updated: %s\n", event.NewValue.Name)
//				case "delete":
//					fmt.Println("User deleted")
//				}
//			}
//		}()
//
//		// Continue with application logic...
//	}
//
// Error Handling:
//
//	import (
//		"fmt"
//		"github.com/Abraxas-365/craftable/errx"
//		"github.com/Abraxas-365/craftable/storex"
//	)
//
//	func ExampleErrorHandling(err error) {
//		// Check for specific error types
//		if storex.IsRecordNotFound(err) {
//			fmt.Println("Record was not found")
//			return
//		}
//
//		if storex.IsConnectionFailed(err) {
//			fmt.Println("Database connection failed, retrying...")
//			return
//		}
//
//		if storex.IsInvalidQuery(err) {
//			fmt.Println("Query is invalid, please check parameters")
//			return
//		}
//
//		// With the errx package, you can also extract details
//		// if err is an errx.Error type from this package
//		if errxErr, ok := err.(errx.Error); ok {
//			fmt.Printf("Error code: %s\n", errxErr.Code())
//			fmt.Printf("Details: %v\n", errxErr.Details())
//			fmt.Printf("Cause: %v\n", errxErr.Cause())
//		} else {
//			fmt.Printf("Unknown error: %v\n", err)
//		}
//	}
package storex
