package asyncx

import (
	"fmt"

	"github.com/Abraxas-365/craftable/errx"
)

// Error registry for asyncx
var ErrorRegistry = errx.NewRegistry("ASYNC")

// Error codes for asyncx
var (
	ErrCanceled      = ErrorRegistry.Register("CANCELED", errx.TypeSystem, 499, "Operation was canceled")
	ErrTimeout       = ErrorRegistry.Register("TIMEOUT", errx.TypeTimeout, 408, "Operation timed out")
	ErrPoolSize      = ErrorRegistry.Register("INVALID_POOL_SIZE", errx.TypeBadRequest, 400, "Invalid pool size")
	ErrBatchSize     = ErrorRegistry.Register("INVALID_BATCH_SIZE", errx.TypeBadRequest, 400, "Invalid batch size")
	ErrRetryAttempts = ErrorRegistry.Register("INVALID_RETRY_ATTEMPTS", errx.TypeBadRequest, 400, "Invalid retry attempts")
	ErrNoFunctions   = ErrorRegistry.Register("NO_FUNCTIONS", errx.TypeBadRequest, 400, "No functions provided")
	ErrAllFailed     = ErrorRegistry.Register("ALL_FAILED", errx.TypeSystem, 500, "All operations failed")
)

// ErrorCollection holds multiple errors from concurrent operations
type ErrorCollection struct {
	Errors    map[int]error
	Operation string
}

// Error implements the error interface
func (e *ErrorCollection) Error() string {
	return fmt.Sprintf("%s: %d operations failed", e.Operation, len(e.Errors))
}

// IsErrorCollection checks if an error is an ErrorCollection
func IsErrorCollection(err error) (*ErrorCollection, bool) {
	ec, ok := err.(*ErrorCollection)
	return ec, ok
}

// HasError checks if there was an error at a specific index
func (e *ErrorCollection) HasError(index int) bool {
	_, exists := e.Errors[index]
	return exists
}

// GetError retrieves the error at a specific index
func (e *ErrorCollection) GetError(index int) error {
	if err, exists := e.Errors[index]; exists {
		return err
	}
	return nil
}

// FilterSuccessful returns only the successful results
func FilterSuccessful[R any](results []R, err error) []R {
	ec, ok := err.(*ErrorCollection)
	if !ok {
		if err != nil {
			return nil
		}
		return results
	}

	successful := make([]R, 0, len(results))
	for i, result := range results {
		if !ec.HasError(i) {
			successful = append(successful, result)
		}
	}
	return successful
}
