package main

import (
	"github.com/Abraxas-365/craftable/docx"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

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

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func main() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	app := fiber.New()

	// Add CORS middleware
	app.Use(cors.New())

	// Create a router documentation
	apiDoc := docx.NewRouterDoc("/api/v1")

	// Document login endpoint
	loginEndpoint := docx.NewEndpoint("/auth/login", docx.POST).
		WithDescription("Authenticate a user and receive access token").
		WithSummary("User Login").
		WithTags("auth").
		WithRequestDTO(LoginRequest{}).
		WithResponseDTO(TokenResponse{}).
		WithAuth(docx.None, nil).
		WithHeader("Content-Type", "application/json", true).
		WithRequestExample(LoginRequest{
			Email:    "john@example.com",
			Password: "secure123",
		}).
		WithResponseExample(TokenResponse{
			Token:        "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			RefreshToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			ExpiresIn:    3600,
		})

	// Document create user endpoint WITH security
	createUserEndpoint := docx.NewEndpoint("/users", docx.POST).
		WithDescription("Create a new user (requires admin privileges)").
		WithSummary("User Registration").
		WithTags("users", "auth").
		WithRequestDTO(UserCreateRequest{}).
		WithResponseDTO(UserResponse{}).
		WithAuth(docx.Bearer, map[string]string{
			"description": "JWT token required",
			"scope":       "admin",
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

	// Document get users endpoint with different security level
	getUsersEndpoint := docx.NewEndpoint("/users", docx.GET).
		WithDescription("Get list of users").
		WithSummary("List Users").
		WithTags("users").
		WithResponseDTO([]UserResponse{}).
		WithAuth(docx.Bearer, map[string]string{
			"description": "JWT token required",
			"scope":       "read:users",
		}).
		WithResponseExample([]UserResponse{
			{
				ID:    "123",
				Name:  "John Doe",
				Email: "john@example.com",
			},
			{
				ID:    "456",
				Name:  "Jane Smith",
				Email: "jane@example.com",
			},
		})

	// Add endpoints to router doc
	apiDoc.AddEndpoint(loginEndpoint)
	apiDoc.AddEndpoint(createUserEndpoint)
	apiDoc.AddEndpoint(getUsersEndpoint)

	// Create a generator
	generator := docx.NewGenerator().AddRouter(apiDoc)

	// Generate JSON documentation
	generator.GenerateJSON("docs/api.json")

	// Generate curl documentation
	generator.GenerateCurlDocs("http://localhost:3000", "docs/curl.md")

	// Register documentation in Fiber app
	apiDoc.RegisterWithFiber(app, "/api/docs")

	// Auth middleware function
	authMiddleware := func(c *fiber.Ctx) error {
		// Get authorization header
		authHeader := c.Get("Authorization")

		// Check if authorization header exists and has the correct format
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing or invalid authorization token",
			})
		}

		// Extract the token
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// In a real application, you would validate the JWT token here
		// For this example, we'll just check if it's not empty
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		// For demonstration, let's add a mock user ID to the context
		// In a real app, you would extract this from the token
		c.Locals("userId", "user_"+strconv.Itoa(rand.Intn(1000)))

		// Continue to the next middleware/handler
		return c.Next()
	}

	// Login endpoint implementation
	app.Post("/api/v1/auth/login", func(c *fiber.Ctx) error {
		// Parse login request
		var loginReq LoginRequest
		if err := c.BodyParser(&loginReq); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request data",
			})
		}

		// In a real app, you would validate credentials against a database
		// For this example, we'll just check if the email contains "admin"
		isAdmin := strings.Contains(loginReq.Email, "admin")

		// Create a mock token (in a real app, you would sign a proper JWT)
		mockToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJyb2xlIjoiYWRtaW4ifQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
		if !isAdmin {
			mockToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJyb2xlIjoidXNlciJ9.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
		}

		// Return token response
		return c.Status(fiber.StatusOK).JSON(TokenResponse{
			Token:        mockToken,
			RefreshToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJyZWZyZXNoIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			ExpiresIn:    3600,
		})
	})

	// Create user endpoint implementation with auth
	app.Post("/api/v1/users", authMiddleware, func(c *fiber.Ctx) error {
		// Parse incoming user data
		var userReq UserCreateRequest
		if err := c.BodyParser(&userReq); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request data",
			})
		}

		// Generate fake ID for testing
		fakeID := strconv.Itoa(rand.Intn(1000))

		// Create response
		userResp := UserResponse{
			ID:    fakeID,
			Name:  userReq.Name,
			Email: userReq.Email,
		}

		// Return success response
		return c.Status(fiber.StatusCreated).JSON(userResp)
	})

	// Get users endpoint with auth
	app.Get("/api/v1/users", authMiddleware, func(c *fiber.Ctx) error {
		// In a real app, you would fetch users from a database
		// Here we'll return mock data
		users := []UserResponse{
			{
				ID:    "123",
				Name:  "John Doe",
				Email: "john@example.com",
			},
			{
				ID:    "456",
				Name:  "Jane Smith",
				Email: "jane@example.com",
			},
		}

		return c.Status(fiber.StatusOK).JSON(users)
	})

	app.Listen(":3000")
}

