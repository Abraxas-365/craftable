// Package dtox provides utilities for converting between DTOs (Data Transfer Objects) and domain models.
//
// The package uses Go generics to provide type-safe conversion between DTO types used in API/presentation
// layers and domain model types used in business logic. It supports field mapping, validation,
// custom conversion functions, and batch operations.
//
// Basic Example:
//
//	import (
//		"github.com/Abraxas-365/craftable/dtox"
//	)
//
//	// Define DTO and model types
//	type UserDTO struct {
//		Name string `json:"name"`
//		Age  int    `json:"age"`
//	}
//
//	type User struct {
//		ID        string
//		FirstName string
//		Age       int
//		CreatedAt time.Time
//	}
//
//	// Create a mapper
//	mapper := dtox.NewMapper[UserDTO, User]().
//		WithFieldMapping("Name", "FirstName")
//
//	// Convert DTO to model
//	dto := UserDTO{Name: "John Doe", Age: 30}
//	user, err := mapper.ToModel(dto)
//	if err != nil {
//		// Handle error
//	}
//
// Advanced Features:
//
// 1. Custom Conversion Functions:
//
//	mapper := dtox.NewMapper[UserDTO, User]().
//		WithCustomDtoToModel(func(dto UserDTO) (User, error) {
//			return User{
//				FirstName: dto.Name,
//				Age:       dto.Age,
//				CreatedAt: time.Now(),
//			}, nil
//		})
//
// 2. Validation:
//
//	mapper := dtox.NewMapper[UserDTO, User]().
//		WithValidation(func(dto UserDTO) error {
//			if dto.Name == "" {
//				return errors.New("name is required")
//			}
//			if dto.Age < 0 {
//				return errors.New("age cannot be negative")
//			}
//			return nil
//		})
//
// 3. Batch Conversion:
//
//	dtos := []UserDTO{
//		{Name: "User 1", Age: 25},
//		{Name: "User 2", Age: 30},
//	}
//
//	// Convert all DTOs to models
//	users, err := mapper.ToModels(dtos)
//
// 4. Ignoring Fields:
//
//	mapper := dtox.NewMapper[UserDTO, User]().
//		WithIgnoreField("Age")  // Age field will not be mapped
package dtox
