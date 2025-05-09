// Package validatex provides struct validation through field tags with error handling via errx.
//
// Validatex allows you to define validation rules directly in your struct
// using field tags, then validate instances of those structs with a simple API.
// It integrates with the errx package to provide structured, consistent error handling.
//
// Basic Usage:
//
//	import (
//		"github.com/Abraxas-365/craftale/errx"
//		"github.com/Abraxas-365/craftale/validatex"
//	)
//
//	type User struct {
//		Username string `validatex:"required,min=3,max=50"`
//		Email    string `validatex:"required,email"`
//		Age      int    `validatex:"min=18,max=120"`
//		Password string `validatex:"required,min=8"`
//		Role     string `validatex:"oneof=admin user guest"`
//	}
//
//	func main() {
//		user := User{
//			Username: "jo", // too short
//			Email:    "not-an-email",
//			Age:      15,   // too young
//			Password: "short",
//			Role:     "superuser", // not in allowed values
//		}
//
//		// Validate the struct
//		if err := validatex.ValidateWithErrx(user); err != nil {
//			// Handle structured error - it's an errx.Error
//			fmt.Println(err.Code, err.Type, err.Message)
//			fmt.Println(err.Details) // Contains field-specific errors
//
//			// Use in HTTP response
//			err.ToHTTP(responseWriter)
//		}
//	}
//
// Available Validation Rules:
//
//   - required: field must not be empty
//   - email: field must be a valid email
//   - url: field must be a valid URL
//   - min=N: minimum value (for numbers) or length (for strings)
//   - max=N: maximum value (for numbers) or length (for strings)
//   - oneof=V1 V2 V3: field must be one of the specified values
//   - regex=PATTERN: field must match the regular expression
//   - uuid: field must be a valid UUID
//   - alphanum: field must contain only alphanumeric characters
//   - alpha: field must contain only alphabetic characters
//   - numeric: field must contain only numeric characters
//
// Custom Validation:
//
// You can register custom validation functions:
//
//	validatex.RegisterValidationFunc("customRule", func(value interface{}, param string) bool {
//		// Implement custom validation logic
//		return true // or false if validation fails
//	})
package validatex
