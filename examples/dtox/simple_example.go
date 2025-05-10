package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/Abraxas-365/craftable/dtox"
	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/craftable/validatex"
)

// Basic example of DTO to Model conversion
func BasicDtoxExample() {
	// Define DTO and model types
	type UserDTO struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	type User struct {
		ID        string
		FirstName string
		Email     string
		Age       int
		CreatedAt time.Time
	}

	// Create a mapper with field mapping
	mapper := dtox.NewMapper[UserDTO, User]().
		WithFieldMapping("Name", "FirstName")

	// Convert a DTO to a model
	dto := UserDTO{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	user, err := mapper.ToModel(dto)
	if err != nil {
		fmt.Printf("Error converting DTO to model: %v\n", err)
		return
	}

	fmt.Println("Basic conversion example:")
	fmt.Printf("User: %+v\n", user)
	fmt.Println("Note: ID and CreatedAt are not set because they're not in the DTO")
	fmt.Println()
}

// Example using field mapping strategies
func FieldMappingStrategies() {
	// Define DTO and model types with different naming conventions
	type ProductDTO struct {
		ProductName  string  `json:"product_name"`
		ProductPrice float64 `json:"product_price"`
		InStock      bool    `json:"in_stock"`
	}

	type Product struct {
		Name     string
		Price    float64
		IsActive bool
	}

	// Create mappers with different field mapping modes
	exactMapper := dtox.NewMapper[ProductDTO, Product]().
		AutoMapFields(dtox.ExactMatch)

	camelMapper := dtox.NewMapper[ProductDTO, Product]().
		AutoMapFields(dtox.SnakeToCamelMatch).
		WithFieldMapping("InStock", "IsActive")

	flexibleMapper := dtox.NewMapper[ProductDTO, Product]().
		AutoMapFields(dtox.FlexibleMatch).
		WithFieldMapping("InStock", "IsActive")

	// Create a DTO to convert
	dto := ProductDTO{
		ProductName:  "Laptop",
		ProductPrice: 999.99,
		InStock:      true,
	}

	// Try different mapping strategies
	fmt.Println("Field mapping strategies:")

	// Exact matching won't work for different naming conventions
	product1, err := exactMapper.ToModel(dto)
	if err != nil {
		fmt.Printf("Exact mapping error: %v\n", err)
	} else {
		fmt.Printf("Exact mapping result: %+v\n", product1)
	}

	// Snake case to camel case mapping
	product2, err := camelMapper.ToModel(dto)
	if err != nil {
		fmt.Printf("Snake-to-camel mapping error: %v\n", err)
	} else {
		fmt.Printf("Snake-to-camel mapping result: %+v\n", product2)
	}

	// Flexible mapping tries multiple strategies
	product3, err := flexibleMapper.ToModel(dto)
	if err != nil {
		fmt.Printf("Flexible mapping error: %v\n", err)
	} else {
		fmt.Printf("Flexible mapping result: %+v\n", product3)
	}
	fmt.Println()
}

// Example with validation
func ValidationExample() {
	// Define DTO and model types
	type UserDTO struct {
		Username string `json:"username" validatex:"required,min=3"`
		Email    string `json:"email" validatex:"required,email"`
		Age      int    `json:"age" validatex:"min=18"`
	}

	type User struct {
		Username  string
		Email     string
		Age       int
		CreatedAt time.Time
	}

	// Create a mapper with validation
	mapper := dtox.NewMapper[UserDTO, User]().
		WithValidation(func(dto UserDTO) error {
			return validatex.Validate(dto)
		})

	// Create valid and invalid DTOs
	validDTO := UserDTO{
		Username: "johndoe",
		Email:    "john@example.com",
		Age:      25,
	}

	invalidDTO := UserDTO{
		Username: "jo", // too short
		Email:    "not-an-email",
		Age:      16, // too young
	}

	fmt.Println("Validation example:")

	// Try converting valid DTO
	user1, err := mapper.ToModel(validDTO)
	if err != nil {
		fmt.Printf("Valid DTO conversion error: %v\n", err)
	} else {
		fmt.Printf("Valid DTO conversion result: %+v\n", user1)
	}

	// Try converting invalid DTO
	user2, err := mapper.ToModel(invalidDTO)
	if err != nil {
		fmt.Printf("Invalid DTO conversion error: %v\n", err)
	} else {
		fmt.Printf("Invalid DTO conversion result: %+v\n", user2)
	}
	fmt.Println()
}

// Example with batch conversion
func BatchConversionExample() {
	// Define DTO and model types
	type ItemDTO struct {
		Name     string  `json:"name"`
		Price    float64 `json:"price"`
		Quantity int     `json:"quantity"`
	}

	type Item struct {
		Name      string
		Price     float64
		Quantity  int
		Total     float64
		CreatedAt time.Time
	}

	// Create a mapper with custom conversion
	mapper := dtox.NewMapper[ItemDTO, Item]().
		WithCustomDtoToModel(func(dto ItemDTO) (Item, error) {
			return Item{
				Name:      dto.Name,
				Price:     dto.Price,
				Quantity:  dto.Quantity,
				Total:     dto.Price * float64(dto.Quantity),
				CreatedAt: time.Now(),
			}, nil
		})

	// Create a batch of DTOs
	dtos := []ItemDTO{
		{Name: "Product 1", Price: 10.99, Quantity: 2},
		{Name: "Product 2", Price: 24.99, Quantity: 1},
		{Name: "Product 3", Price: 5.99, Quantity: 5},
	}

	// Convert all DTOs to models
	items, err := mapper.ToModels(dtos)
	if err != nil {
		fmt.Printf("Batch conversion error: %v\n", err)
		return
	}

	fmt.Println("Batch conversion example:")
	for i, item := range items {
		fmt.Printf("Item %d: %+v\n", i+1, item)
	}
	fmt.Println()
}

// Example with partial updates
func PartialUpdateExample() {
	// Define DTO and model types
	type UserUpdateDTO struct {
		Email    string `json:"email,omitempty"`
		Username string `json:"username,omitempty"`
		Age      int    `json:"age,omitempty"`
	}

	type User struct {
		ID       string
		Email    string
		Username string
		Age      int
		IsActive bool
	}

	// Create existing user
	existingUser := User{
		ID:       "user-123",
		Email:    "old@example.com",
		Username: "olduser",
		Age:      30,
		IsActive: true,
	}

	// Create a mapper for partial updates
	mapper := dtox.NewMapper[UserUpdateDTO, User]().
		WithPartial(true)

	// Create a partial update DTO with only some fields set
	updateDTO := UserUpdateDTO{
		Email: "new@example.com",
		// Username and Age are not provided, so they shouldn't be updated
	}

	// Apply the partial update
	updatedUser, err := mapper.ApplyPartialUpdate(existingUser, updateDTO)
	if err != nil {
		fmt.Printf("Partial update error: %v\n", err)
		return
	}

	fmt.Println("Partial update example:")
	fmt.Printf("Original user: %+v\n", existingUser)
	fmt.Printf("Updated user: %+v\n", updatedUser)
	fmt.Println("Note: Only Email was updated, other fields retained their values")
	fmt.Println()
}

// Example with custom options
func CustomOptionsExample() {
	// Define DTO and model types
	type ProductDTO struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}

	type Product struct {
		ID          string
		Name        string
		Description string
		Price       float64
		UpdatedAt   time.Time
	}

	// Create custom options
	options := dtox.NewOptions().
		WithParallelProcessing(true).
		WithParallelThreshold(50).
		WithMaxConcurrency(4).
		WithFieldMappingMode(dtox.FlexibleMatch)

	// Create a mapper with custom options
	mapper := dtox.NewMapper[ProductDTO, Product]().
		WithOptions(options).
		WithCustomDtoToModel(func(dto ProductDTO) (Product, error) {
			return Product{
				ID:          fmt.Sprintf("prod-%d", time.Now().UnixNano()),
				Name:        dto.Name,
				Description: dto.Description,
				Price:       dto.Price,
				UpdatedAt:   time.Now(),
			}, nil
		})

	// Create a DTO
	dto := ProductDTO{
		Name:        "Smart TV",
		Description: "55-inch 4K Smart TV",
		Price:       499.99,
	}

	// Convert to model
	product, err := mapper.ToModel(dto)
	if err != nil {
		fmt.Printf("Conversion error: %v\n", err)
		return
	}

	fmt.Println("Custom options example:")
	fmt.Printf("Product: %+v\n", product)
	fmt.Println()
}

// Example with errx integration
func ErrxIntegrationExample() {
	// Define DTO and model types
	type UserDTO struct {
		Username string `json:"username" validatex:"required,min=3"`
		Email    string `json:"email" validatex:"required,email"`
	}

	type User struct {
		Username string
		Email    string
	}

	// Create a validation function that returns errx errors
	validateUser := func(dto UserDTO) error {
		if err := validatex.ValidateWithErrx(dto); err != nil {
			return err // Already an errx.Error
		}
		return nil
	}

	// Create a mapper with validation
	mapper := dtox.NewMapper[UserDTO, User]().
		WithValidation(validateUser).
		WithStrictMode(true)

	// Create an invalid DTO
	invalidDTO := UserDTO{
		Username: "u", // too short
		Email:    "invalid-email",
	}

	// Try converting
	_, err := mapper.ToModel(invalidDTO)

	fmt.Println("Errx integration example:")
	if err != nil {
		// Check if it's an errx.Error
		var xerr *errx.Error
		if errors.As(err, &xerr) {
			fmt.Printf("Validation error detected:\n")
			fmt.Printf("  Code: %s\n", xerr.Code)
			fmt.Printf("  Type: %s\n", xerr.Type)
			fmt.Printf("  Message: %s\n", xerr.Message)
			fmt.Printf("  Details: %+v\n", xerr.Details)

			// In a real API handler, you would use:
			// xerr.ToHTTP(responseWriter)
		} else {
			fmt.Printf("Unexpected error type: %v\n", err)
		}
	} else {
		fmt.Println("Conversion succeeded unexpectedly")
	}
	fmt.Println()
}

// Example with bidirectional conversion
func BidirectionalExample() {
	// Define DTO and model types
	type AddressDTO struct {
		Street  string `json:"street"`
		City    string `json:"city"`
		Country string `json:"country"`
	}

	type Address struct {
		Street     string
		City       string
		Country    string
		IsVerified bool
	}

	// Create a mapper with bidirectional conversion
	mapper := dtox.NewMapper[AddressDTO, Address]().
		WithCustomDtoToModel(func(dto AddressDTO) (Address, error) {
			return Address{
				Street:     dto.Street,
				City:       dto.City,
				Country:    dto.Country,
				IsVerified: false, // Default for new addresses
			}, nil
		}).
		WithCustomModelToDto(func(model Address) (AddressDTO, error) {
			return AddressDTO{
				Street:  model.Street,
				City:    model.City,
				Country: model.Country,
				// IsVerified isn't included in the DTO
			}, nil
		})

	// DTO to Model
	dto := AddressDTO{
		Street:  "123 Main St",
		City:    "New York",
		Country: "USA",
	}

	model, err := mapper.ToModel(dto)
	if err != nil {
		fmt.Printf("DTO to model error: %v\n", err)
		return
	}

	// Verify the address
	model.IsVerified = true

	// Model back to DTO
	dtoResult, err := mapper.ToDto(model)
	if err != nil {
		fmt.Printf("Model to DTO error: %v\n", err)
		return
	}

	fmt.Println("Bidirectional conversion example:")
	fmt.Printf("Original DTO: %+v\n", dto)
	fmt.Printf("Model: %+v\n", model)
	fmt.Printf("Result DTO: %+v\n", dtoResult)
	fmt.Println()
}

// RunAllDtoxExamples runs all the dtox examples
func main() {
	fmt.Println("========= DTOX EXAMPLES =========")

	BasicDtoxExample()
	FieldMappingStrategies()
	ValidationExample()
	BatchConversionExample()
	PartialUpdateExample()
	CustomOptionsExample()
	ErrxIntegrationExample()
	BidirectionalExample()

	fmt.Println("All dtox examples completed.")
}
