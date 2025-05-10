package asyncx

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/Abraxas-365/craftable/errx"
)

// AsyncAll runs the provided function on each item in parallel and returns all results
// Returns on first error
func AsyncAll[T any, R any](ctx context.Context, items []T, fn func(ctx context.Context, item T) (R, error)) ([]R, error) {
	var wg sync.WaitGroup
	results := make([]R, len(items))

	// Create a cancellable context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Error channel with buffer size 1
	errCh := make(chan error, 1)

	for i, item := range items {
		wg.Add(1)
		go func(idx int, it T) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			result, err := fn(ctx, it)
			if err != nil {
				// Try to send error - non-blocking
				select {
				case errCh <- err:
					// Successfully sent error, cancel other operations
					cancel()
				default:
					// Error channel already has an error
				}
				return
			}

			results[idx] = result
		}(i, item)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Check for errors
	select {
	case err := <-errCh:
		return nil, errx.Wrap(err, "async operation failed", errx.TypeSystem)
	case <-ctx.Done():
		if ctx.Err() == context.Canceled {
			return nil, ErrorRegistry.NewWithCause(ErrCanceled, ctx.Err())
		}
		return nil, ErrorRegistry.NewWithCause(ErrTimeout, ctx.Err())
	default:
		return results, nil
	}
}

// AsyncAllCollect executes all functions concurrently and collects all errors
// Similar to Rust's approach of collecting errors
func AsyncAllCollect[T any, R any](ctx context.Context, items []T, fn func(ctx context.Context, item T) (R, error)) ([]R, error) {
	var wg sync.WaitGroup
	results := make([]R, len(items))
	var mu sync.Mutex
	errorMap := make(map[int]error)

	for i, item := range items {
		wg.Add(1)
		go func(idx int, itm T) {
			defer wg.Done()

			// Check if context is already cancelled
			select {
			case <-ctx.Done():
				mu.Lock()
				if ctx.Err() == context.Canceled {
					errorMap[idx] = ErrorRegistry.NewWithCause(ErrCanceled, ctx.Err())
				} else {
					errorMap[idx] = ErrorRegistry.NewWithCause(ErrTimeout, ctx.Err())
				}
				mu.Unlock()
				return
			default:
			}

			result, err := fn(ctx, itm)
			if err != nil {
				mu.Lock()
				errorMap[idx] = err
				mu.Unlock()
				return
			}

			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, item)
	}

	wg.Wait()

	// Return collected errors if any
	if len(errorMap) > 0 {
		return results, &ErrorCollection{
			Errors:    errorMap,
			Operation: "AsyncAllCollect",
		}
	}

	return results, nil
}

// AsyncPool executes async operations with limited concurrency
func AsyncPool[T any, R any](ctx context.Context, items []T, concurrency int,
	fn func(ctx context.Context, item T) (R, error)) ([]R, error) {

	if concurrency <= 0 {
		return nil, ErrorRegistry.New(ErrPoolSize).WithDetail("provided", concurrency)
	}

	if concurrency > len(items) {
		concurrency = len(items)
	}

	var wg sync.WaitGroup
	results := make([]R, len(items))

	// Create a cancellable context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Error channel
	errCh := make(chan error, 1)

	// Channel to control concurrency
	semaphore := make(chan struct{}, concurrency)

	for i, item := range items {
		// Acquire semaphore slot
		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				return nil, ErrorRegistry.NewWithCause(ErrCanceled, ctx.Err())
			}
			return nil, ErrorRegistry.NewWithCause(ErrTimeout, ctx.Err())
		}

		wg.Add(1)
		go func(idx int, it T) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore slot

			select {
			case <-ctx.Done():
				return
			default:
			}

			result, err := fn(ctx, it)
			if err != nil {
				select {
				case errCh <- err:
					cancel()
				default:
				}
				return
			}

			results[idx] = result
		}(i, item)
	}

	wg.Wait()

	select {
	case err := <-errCh:
		return nil, errx.Wrap(err, "pool operation failed", errx.TypeSystem)
	case <-ctx.Done():
		if ctx.Err() == context.Canceled {
			return nil, ErrorRegistry.NewWithCause(ErrCanceled, ctx.Err())
		}
		return nil, ErrorRegistry.NewWithCause(ErrTimeout, ctx.Err())
	default:
		return results, nil
	}
}

// AsyncPoolCollect executes operations with limited concurrency and collects all errors
func AsyncPoolCollect[T any, R any](ctx context.Context, items []T, concurrency int,
	fn func(ctx context.Context, item T) (R, error)) ([]R, error) {

	if concurrency <= 0 {
		return nil, ErrorRegistry.New(ErrPoolSize).WithDetail("provided", concurrency)
	}

	if concurrency > len(items) {
		concurrency = len(items)
	}

	var wg sync.WaitGroup
	results := make([]R, len(items))
	var mu sync.Mutex
	errorMap := make(map[int]error)

	// Channel to control concurrency
	semaphore := make(chan struct{}, concurrency)

	for i, item := range items {
		// Acquire semaphore slot
		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				return nil, ErrorRegistry.NewWithCause(ErrCanceled, ctx.Err())
			}
			return nil, ErrorRegistry.NewWithCause(ErrTimeout, ctx.Err())
		}

		wg.Add(1)
		go func(idx int, it T) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore slot

			select {
			case <-ctx.Done():
				mu.Lock()
				if ctx.Err() == context.Canceled {
					errorMap[idx] = ErrorRegistry.NewWithCause(ErrCanceled, ctx.Err())
				} else {
					errorMap[idx] = ErrorRegistry.NewWithCause(ErrTimeout, ctx.Err())
				}
				mu.Unlock()
				return
			default:
			}

			result, err := fn(ctx, it)
			if err != nil {
				mu.Lock()
				errorMap[idx] = err
				mu.Unlock()
				return
			}

			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, item)
	}

	wg.Wait()

	// Return collected errors if any
	if len(errorMap) > 0 {
		return results, &ErrorCollection{
			Errors:    errorMap,
			Operation: "AsyncPoolCollect",
		}
	}

	return results, nil
}

// AsyncBatch processes items in batches with each batch running concurrently
func AsyncBatch[T any, R any](ctx context.Context, items []T, batchSize int,
	fn func(ctx context.Context, batch []T) ([]R, error)) ([]R, error) {

	if batchSize <= 0 {
		return nil, ErrorRegistry.New(ErrBatchSize).WithDetail("provided", batchSize)
	}

	// Create batches
	var batches [][]T
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}

	var resultsMutex sync.Mutex
	var allResults []R

	// Process each batch
	_, err := AsyncAll(ctx, batches, func(ctx context.Context, batch []T) (struct{}, error) {
		batchResults, err := fn(ctx, batch)
		if err != nil {
			return struct{}{}, err
		}

		resultsMutex.Lock()
		allResults = append(allResults, batchResults...)
		resultsMutex.Unlock()

		return struct{}{}, nil
	})

	if err != nil {
		return nil, err
	}

	return allResults, nil
}

// AsyncBatchCollect processes items in batches and collects all errors
func AsyncBatchCollect[T any, R any](ctx context.Context, items []T, batchSize int,
	fn func(ctx context.Context, batch []T) ([]R, error)) ([]R, error) {

	if batchSize <= 0 {
		return nil, ErrorRegistry.New(ErrBatchSize).WithDetail("provided", batchSize)
	}

	// Create batches
	var batches [][]T
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}

	var resultsMutex sync.Mutex
	var allResults []R
	var errorMutex sync.Mutex
	errorMap := make(map[int]error)

	// Process each batch
	for i, batch := range batches {
		err := func(batchIdx int, batchItems []T) error {
			batchResults, err := fn(ctx, batchItems)
			if err != nil {
				errorMutex.Lock()
				errorMap[batchIdx] = err
				errorMutex.Unlock()
				return err
			}

			resultsMutex.Lock()
			allResults = append(allResults, batchResults...)
			resultsMutex.Unlock()

			return nil
		}(i, batch)

		if err != nil && ctx.Err() != nil {
			// If context is canceled or timed out, stop processing
			break
		}
	}

	// Return collected errors if any
	if len(errorMap) > 0 {
		return allResults, &ErrorCollection{
			Errors:    errorMap,
			Operation: "AsyncBatchCollect",
		}
	}

	return allResults, nil
}

// Map applies a function to each item concurrently and returns all results
func Map[T any, R any](ctx context.Context, items []T, concurrency int,
	mapFn func(ctx context.Context, item T) (R, error)) ([]R, error) {
	return AsyncPool(ctx, items, concurrency, mapFn)
}

// MapCollect applies a function to all items and collects all errors
func MapCollect[T any, R any](ctx context.Context, items []T, concurrency int,
	mapFn func(ctx context.Context, item T) (R, error)) ([]R, error) {
	return AsyncPoolCollect(ctx, items, concurrency, mapFn)
}

// Filter concurrently keeps items that pass the filter function
func Filter[T any](ctx context.Context, items []T, concurrency int,
	filterFn func(ctx context.Context, item T) (bool, error)) ([]T, error) {
	type indexedItem struct {
		idx  int
		keep bool
	}

	results, err := AsyncPool(ctx, items, concurrency,
		func(ctx context.Context, item T) (indexedItem, error) {
			keep, err := filterFn(ctx, item)
			return indexedItem{idx: 0, keep: keep}, err
		})

	if err != nil {
		return nil, err
	}

	// Build result slice
	var filtered []T
	for i, result := range results {
		if result.keep {
			filtered = append(filtered, items[i])
		}
	}

	return filtered, nil
}

// FilterCollect concurrently filters items and collects all errors
func FilterCollect[T any](ctx context.Context, items []T, concurrency int,
	filterFn func(ctx context.Context, item T) (bool, error)) ([]T, *ErrorCollection) {
	type indexedItem struct {
		idx  int
		keep bool
	}

	results, err := AsyncPoolCollect(ctx, items, concurrency,
		func(ctx context.Context, item T) (indexedItem, error) {
			keep, err := filterFn(ctx, item)
			return indexedItem{idx: 0, keep: keep}, err
		})

	// Build result slice regardless of errors
	var filtered []T
	for i, result := range results {
		if result.keep {
			filtered = append(filtered, items[i])
		}
	}

	// Return errors if any
	if ec, ok := err.(*ErrorCollection); ok {
		return filtered, ec
	}

	if err != nil {
		return nil, &ErrorCollection{
			Errors:    map[int]error{0: err},
			Operation: "FilterCollect",
		}
	}

	return filtered, nil
}

// RetryWithBackoff retries a function with exponential backoff
func RetryWithBackoff[T any](ctx context.Context, attempts int, initialBackoff time.Duration,
	fn func(ctx context.Context) (T, error)) (T, error) {

	if attempts <= 0 {
		var zero T
		return zero, ErrorRegistry.New(ErrRetryAttempts).WithDetail("provided", attempts)
	}

	var result T
	var lastErr error

	backoff := initialBackoff
	for i := 0; i < attempts; i++ {
		select {
		case <-ctx.Done():
			var zero T
			if ctx.Err() == context.Canceled {
				return zero, ErrorRegistry.NewWithCause(ErrCanceled, ctx.Err())
			}
			return zero, ErrorRegistry.NewWithCause(ErrTimeout, ctx.Err())
		default:
		}

		result, lastErr = fn(ctx)
		if lastErr == nil {
			return result, nil
		}

		if i == attempts-1 {
			break
		}

		// Add jitter to backoff (10% variance)
		jitter := time.Duration(rand.Float64() * 0.1 * float64(backoff))
		actualBackoff := backoff + jitter

		select {
		case <-time.After(actualBackoff):
			backoff = backoff * 2 // Exponential backoff
		case <-ctx.Done():
			var zero T
			if ctx.Err() == context.Canceled {
				return zero, ErrorRegistry.NewWithCause(ErrCanceled, ctx.Err())
			}
			return zero, ErrorRegistry.NewWithCause(ErrTimeout, ctx.Err())
		}
	}

	var zero T
	return zero, errx.Wrap(lastErr, "all retry attempts failed", errx.TypeSystem)
}

// FirstSuccess runs multiple functions concurrently and returns the first successful result
func FirstSuccess[T any](ctx context.Context, fns []func(ctx context.Context) (T, error)) (T, error) {
	if len(fns) == 0 {
		var zero T
		return zero, ErrorRegistry.New(ErrNoFunctions)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	resultCh := make(chan T, 1)
	errCh := make(chan error, len(fns))

	var wg sync.WaitGroup
	for _, fn := range fns {
		wg.Add(1)
		go func(f func(ctx context.Context) (T, error)) {
			defer wg.Done()

			result, err := f(ctx)
			if err == nil {
				select {
				case resultCh <- result:
					cancel() // Cancel other operations when we get a success
				default:
					// Result channel already has a value
				}
			} else {
				errCh <- err
			}
		}(fn)
	}

	// Wait in a goroutine so we don't block on checking channels
	go func() {
		wg.Wait()
		close(resultCh)
		close(errCh)
	}()

	// Check for a successful result
	select {
	case result, ok := <-resultCh:
		if ok {
			return result, nil
		}
	case <-ctx.Done():
		if ctx.Err() == context.Canceled {
			// If context was explicitly canceled, a result should be available
			select {
			case result := <-resultCh:
				return result, nil
			default:
				var zero T
				return zero, ErrorRegistry.NewWithCause(ErrCanceled, ctx.Err())
			}
		}
		var zero T
		return zero, ErrorRegistry.NewWithCause(ErrTimeout, ctx.Err())
	}

	// If we're here, no success was found, collect errors
	var allErrors []error
	for err := range errCh {
		allErrors = append(allErrors, err)
	}

	var zero T
	if len(allErrors) > 0 {
		details := make(map[string]any)
		for i, err := range allErrors {
			details[fmt.Sprintf("error_%d", i)] = err.Error()
		}
		return zero, ErrorRegistry.New(ErrAllFailed).WithDetails(details)
	}

	return zero, errx.New("no successful results and no errors reported", errx.TypeSystem)
}
