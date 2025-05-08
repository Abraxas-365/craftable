/*
Package errx provides an extended error handling system for Go applications.
It supports structured errors with types, codes, details, HTTP status mapping,
error wrapping, and integrations with both standard net/http and Fiber.

# Basic Usage

Create simple errors with the New function:

	err := errx.New("item not found", errx.TypeNotFound)

	// Check error type
	if errx.IsType(err, errx.TypeNotFound) {
		// Handle not found case
	}

# Error Registry

For domain-specific errors, create a registry with prefixed error codes:

	// Create a registry for user-related errors
	userErrors := errx.NewRegistry("USER")

	// Register common errors
	ErrUserNotFound := userErrors.Register("NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "User not found")
	ErrInvalidPassword := userErrors.Register("INVALID_PASSWORD", errx.TypeValidation, http.StatusBadRequest, "Invalid password")

	// Create instances of registered errors
	err := userErrors.New(ErrUserNotFound)

	// Create with custom message
	err := userErrors.NewWithMessage(ErrUserNotFound, "User with ID 123 not found")

# Adding Details

Provide additional context with details:

	err := userErrors.New(ErrUserNotFound).
		WithDetail("user_id", "123").
		WithDetail("request_id", requestID)

	// Or with a map
	err := userErrors.New(ErrInvalidPassword).
		WithDetails(map[string]any{
			"field": "password",
			"reason": "too_short",
			"min_length": 8,
		})

# Error Wrapping

Wrap standard errors to add context while preserving the original cause:

	originalErr := sql.ErrNoRows
	err := errx.Wrap(originalErr, "Failed to retrieve user record", errx.TypeNotFound)

	// Or when using a registry
	err := userErrors.NewWithCause(ErrUserNotFound, originalErr)

	// The original error can be retrieved
	var sqlErr *sql.ErrNoRows
	if errors.As(err, &sqlErr) {
		// Handle SQL-specific error
	}

# HTTP Integration

Return structured errors in HTTP handlers:

	// Using standard net/http
	func UserHandler(w http.ResponseWriter, r *http.Request) {
		user, err := findUser(r.PathValue("id"))
		if err != nil {
			if errx.IsType(err, errx.TypeNotFound) {
				e := errx.Wrap(err, "User not found", errx.TypeNotFound)
				e.HTTPStatus = http.StatusNotFound
				e.ToHTTP(w)
				return
			}

			// Handle other errors
			e := errx.Wrap(err, "Internal server error", errx.TypeInternal)
			e.HTTPStatus = http.StatusInternalServerError
			e.ToHTTP(w)
			return
		}

		// Continue with success response...
	}

	// Using Fiber
	func UserHandlerFiber(c *fiber.Ctx) error {
		user, err := findUser(c.Params("id"))
		if err != nil {
			if errx.IsType(err, errx.TypeNotFound) {
				return userErrors.New(ErrUserNotFound).ToFiber()
			}

			// Handle other errors
			return errx.Wrap(err, "Internal server error", errx.TypeInternal).ToFiber()
		}

		// Continue with success response...
		return c.JSON(user)
	}

# Client-Side Error Handling

Parse error responses from APIs:

	resp, err := http.Get("https://api.example.com/users/123")
	if err != nil {
		return nil, errx.Wrap(err, "Failed to call API", errx.TypeExternal)
	}

	if resp.StatusCode >= 400 {
		err := errx.FromResponse(resp)
		return nil, err
	}

	// Process successful response...

# Error Checking

Check for specific error conditions:

	if errx.IsCode(err, ErrUserNotFound) {
		// Handle specific error code
	}

	if errx.IsType(err, errx.TypeValidation) {
		// Handle validation errors
	}

	if errx.IsHTTPStatus(err, http.StatusNotFound) {
		// Handle 404 errors
	}
*/
package errx
