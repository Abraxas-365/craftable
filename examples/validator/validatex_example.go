package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/Abraxas-365/craftale/errx"
	"github.com/Abraxas-365/craftale/validatex"
)

// Basic example demonstrating simple validation rules
func BasicValidation() {
	// Define a struct with validation rules
	type User struct {
		Username string `validatex:"required,min=3,max=50"`
		Email    string `validatex:"required,email"`
		Age      int    `validatex:"min=18,max=120"`
		Password string `validatex:"required,min=8"`
		Role     string `validatex:"oneof=admin user guest"`
	}

	// Create a user with validation errors
	user := User{
		Username: "jo", // too short
		Email:    "not-an-email",
		Age:      15, // too young
		Password: "short",
		Role:     "superuser", // not in allowed values
	}

	// Validate the struct and handle errors
	if err := validatex.ValidateWithErrx(user); err != nil {
		// err is an errx.Error type with structured field validation errors
		fmt.Printf("Validation failed: %s\n", err.Message)

		// Access detailed validation errors
		for field, details := range err.Details {
			fmt.Printf("Field %s: %v\n", field, details)
		}

		// You can use this error directly in HTTP responses
		// err.ToHTTP(responseWriter)
	}
}

// Example with custom error messages
func CustomErrorMessages() {
	// Register some custom error messages
	validatex.SetCustomErrorMessage("required", "This field cannot be empty")
	validatex.SetCustomErrorMessage("email", "Must be a valid email address")

	type ContactForm struct {
		Name    string `validatex:"required" errx_msg:"Your name is required"`
		Email   string `validatex:"required,email" errx_msg:"We need your correct email to contact you"`
		Message string `validatex:"required,min=10" errx_msg:"Please provide a detailed message (at least 10 characters)"`
	}

	form := ContactForm{
		Name:    "",
		Email:   "invalid-email",
		Message: "Hi",
	}

	if err := validatex.ValidateWithErrx(form); err != nil {
		// The error messages will use the custom messages
		fmt.Printf("Form validation failed: %s\n", err.Message)
		for field, details := range err.Details {
			fmt.Printf("Field %s: %v\n", field, details)
		}
	}
}

// Example with nested struct validation
func NestedStructValidation() {
	type Address struct {
		Street  string `validatex:"required"`
		City    string `validatex:"required"`
		Country string `validatex:"required,oneof=USA Canada UK Australia"`
		ZipCode string `validatex:"required,regex=^\\d{5}(-\\d{4})?$"`
	}

	type Customer struct {
		Name    string  `validatex:"required"`
		Email   string  `validatex:"required,email"`
		Address Address `validatex:"required"` // Will recursively validate the Address struct
	}

	customer := Customer{
		Name:  "John Doe",
		Email: "john@example.com",
		Address: Address{
			Street:  "123 Main St",
			City:    "",        // Missing required field
			Country: "Germany", // Not in allowed values
			ZipCode: "ABC",     // Doesn't match regex
		},
	}

	if err := validatex.ValidateWithErrx(customer); err != nil {
		fmt.Printf("Customer validation failed: %s\n", err.Message)
		for field, details := range err.Details {
			fmt.Printf("Field %s: %v\n", field, details)
		}
	}
}

// Example with slice validation
func SliceValidation() {
	type Product struct {
		Name  string  `validatex:"required"`
		Price float64 `validatex:"min=0.01"`
	}

	type Order struct {
		OrderID     string    `validatex:"required,uuid"`
		Products    []Product `validatex:"required,min=1"` // Validate the slice has items
		TotalAmount float64   `validatex:"min=0"`
		CreatedAt   time.Time `validatex:"required"`
	}

	order := Order{
		OrderID:     "not-a-uuid",
		Products:    []Product{}, // Empty slice
		TotalAmount: 0,
		CreatedAt:   time.Time{}, // Zero time
	}

	if err := validatex.ValidateWithErrx(order); err != nil {
		fmt.Printf("Order validation failed: %s\n", err.Message)
		for field, details := range err.Details {
			fmt.Printf("Field %s: %v\n", field, details)
		}
	}
}

// Example with custom validation functions
func CustomValidationFunctions() {
	// Register a custom validation function for strong passwords
	validatex.RegisterValidationFunc("strongpassword", func(value interface{}, param string) bool {
		password, ok := value.(string)
		if !ok {
			return false
		}

		// Password must have at least 8 characters, 1 uppercase, 1 lowercase, 1 number, and 1 special char
		hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
		hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
		hasNumber := regexp.MustCompile(`\d`).MatchString(password)
		hasSpecial := regexp.MustCompile(`[!@#$%^&*()]`).MatchString(password)

		return len(password) >= 8 && hasUpper && hasLower && hasNumber && hasSpecial
	})

	// Register custom error message for the new validation
	validatex.SetCustomErrorMessage("strongpassword", "Password must be at least 8 characters and include uppercase, lowercase, number, and special character")

	type UserCredentials struct {
		Username string `validatex:"required,min=3"`
		Password string `validatex:"required,strongpassword"`
	}

	creds := UserCredentials{
		Username: "johndoe",
		Password: "password123", // Missing uppercase and special char
	}

	if err := validatex.ValidateWithErrx(creds); err != nil {
		fmt.Printf("Credentials validation failed: %s\n", err.Message)
		for field, details := range err.Details {
			fmt.Printf("Field %s: %v\n", field, details)
		}
	}
}

// Example using validatex with HTTP handler
func HTTPHandlerExample(w http.ResponseWriter, r *http.Request) {
	// Define a request payload with validation rules
	type CreateUserRequest struct {
		Username  string `json:"username" validatex:"required,min=3,max=50"`
		Email     string `json:"email" validatex:"required,email"`
		FirstName string `json:"first_name" validatex:"required"`
		LastName  string `json:"last_name" validatex:"required"`
		Age       int    `json:"age" validatex:"min=18"`
	}

	// Parse request body
	var req CreateUserRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		// Handle JSON decode error
		errx.New("Invalid JSON payload", errx.TypeBadRequest).
			ToHTTP(w)
		return
	}

	// Validate the request
	if err := validatex.ValidateWithErrx(req); err != nil {
		// err is already an errx.Error, so we can use it directly
		err.ToHTTP(w)
		return
	}

	// If validation passes, process the request
	// ...

	// Return success response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "User created successfully",
	})
}

// Example with conditional validation
func ConditionalValidation() {
	// Register a custom validator for conditional validation
	validatex.RegisterValidationFunc("requiredif", func(value interface{}, param string) bool {
		// Implementation of conditional validation logic
		// param could be something like "OtherField=value"
		return true // Return validation result
	})

	type ShippingInfo struct {
		DeliveryMethod string `validatex:"required,oneof=pickup shipping"`
		Address        string `validatex:"requiredif=DeliveryMethod=shipping"`
		City           string `validatex:"requiredif=DeliveryMethod=shipping"`
		ZipCode        string `validatex:"requiredif=DeliveryMethod=shipping"`
	}

	// Example with conditional validation failure
	shipping := ShippingInfo{
		DeliveryMethod: "shipping",
		// Address, City, and ZipCode are required when DeliveryMethod is "shipping"
	}

	if err := validatex.ValidateWithErrx(shipping); err != nil {
		fmt.Printf("Shipping validation failed: %s\n", err.Message)
		for field, details := range err.Details {
			fmt.Printf("Field %s: %v\n", field, details)
		}
	}
}

// RunAllExamples runs all the example functions
func main() {
	fmt.Println("=== Running Basic Validation Example ===")
	BasicValidation()

	fmt.Println("\n=== Running Custom Error Messages Example ===")
	CustomErrorMessages()

	fmt.Println("\n=== Running Nested Struct Validation Example ===")
	NestedStructValidation()

	fmt.Println("\n=== Running Slice Validation Example ===")
	SliceValidation()

	fmt.Println("\n=== Running Custom Validation Functions Example ===")
	CustomValidationFunctions()

	fmt.Println("\n=== Running Conditional Validation Example ===")
	ConditionalValidation()

	fmt.Println("\nAll examples completed.")
}
