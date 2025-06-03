package errxfiber

import (
	"errors"
	"log"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/gofiber/fiber/v2"
)

// ToFiber converts the Error to a Fiber error response (for API contexts)
func ErrxToFiber(e *errx.Error) error {
	return fiber.NewError(e.HTTPStatus, e.Message)
}

// FiberErrorHandler creates a middleware that handles Error responses
func FiberErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// Log the error
		log.Printf("Error handling request: %v", err)

		// Check if it's our Error type
		var xerr *errx.Error
		if errors.As(err, &xerr) {
			// Build error response
			errorResponse := fiber.Map{
				"code":    xerr.Code,
				"type":    xerr.Type,
				"message": xerr.Message,
			}

			// Only include details if they exist and are not empty
			if xerr.Details != nil && len(xerr.Details) > 0 {
				errorResponse["details"] = xerr.Details
				// Log additional context
				log.Printf("Error details: %v", xerr.Details)
			}

			// Use default 500 status if HTTPStatus is not set
			status := xerr.HTTPStatus
			if status == 0 {
				status = fiber.StatusInternalServerError
			}

			return c.Status(status).JSON(fiber.Map{
				"error": errorResponse,
			})
		}

		// Handle fiber.Error
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			return c.Status(fiberErr.Code).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "FIBER_ERROR",
					"type":    errx.TypeInternal,
					"message": fiberErr.Message,
				},
			})
		}

		// Handle unknown errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"type":    errx.TypeInternal,
				"message": err.Error(),
			},
		})
	}
}

// Usage example for the documentation:

/*
# Fiber Middleware Integration

Register the error handler with your Fiber app:

	app := fiber.New(fiber.Config{
		ErrorHandler: errxfiber.FiberErrorHandler(),
	})

This will automatically format any errx.Error returned from your handlers
into a consistent JSON response:

	{
		"error": {
			"code": "USER_NOT_FOUND",
			"type": "NOT_FOUND",
			"message": "User with ID 123 not found",
			"details": {
				"user_id": "123",
				"request_id": "abc-123-xyz"
			}
		}
	}

Example handler using the middleware:

	app.Get("/users/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		user, err := userService.FindByID(id)

		if err != nil {
			// This error will be automatically formatted by the middleware
			return userErrors.New(ErrUserNotFound).
				WithDetail("user_id", id).
				WithDetail("request_id", c.GetRespHeader("X-Request-ID"))
		}

		return c.JSON(user)
	})
*/
