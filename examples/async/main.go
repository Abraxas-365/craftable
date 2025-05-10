package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/Abraxas-365/craftable/asyncx"
)

func main() {
	// Set up a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("=== AsyncAll Example ===")
	asyncAllExample(ctx)

	fmt.Println("\n=== AsyncAllCollect Example ===")
	asyncAllCollectExample(ctx)

	fmt.Println("\n=== AsyncPool Example ===")
	asyncPoolExample(ctx)

	fmt.Println("\n=== AsyncPoolCollect Example ===")
	asyncPoolCollectExample(ctx)

	fmt.Println("\n=== AsyncBatch Example ===")
	asyncBatchExample(ctx)

	fmt.Println("\n=== AsyncBatchCollect Example ===")
	asyncBatchCollectExample(ctx)

	fmt.Println("\n=== Map Example ===")
	mapExample(ctx)

	fmt.Println("\n=== MapCollect Example ===")
	mapCollectExample(ctx)

	fmt.Println("\n=== Filter Example ===")
	filterExample(ctx)

	fmt.Println("\n=== FilterCollect Example ===")
	filterCollectExample(ctx)

	fmt.Println("\n=== RetryWithBackoff Example ===")
	retryWithBackoffExample(ctx)

	fmt.Println("\n=== FirstSuccess Example ===")
	firstSuccessExample(ctx)

	fmt.Println("\n=== Real-world Example: Concurrent API fetching ===")
	realWorldExample(ctx)
}

// Example worker function that simulates API processing
func processItem(ctx context.Context, item int) (string, error) {
	// Simulate random processing time
	sleepTime := time.Duration(50+rand.Intn(200)) * time.Millisecond

	select {
	case <-time.After(sleepTime):
		// Simulate random failures (10% chance of failure)
		if rand.Intn(10) == 0 {
			return "", fmt.Errorf("error processing item %d", item)
		}
		return fmt.Sprintf("Processed-%d", item), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// AsyncAll Example
func asyncAllExample(ctx context.Context) {
	items := []int{1, 2, 3, 4, 5}

	results, err := asyncx.AsyncAll(ctx, items, processItem)
	if err != nil {
		log.Printf("AsyncAll failed: %v", err)
		return
	}

	fmt.Println("AsyncAll results:", results)
}

// AsyncAllCollect Example
func asyncAllCollectExample(ctx context.Context) {
	items := []int{1, 2, 3, 4, 5}

	results, err := asyncx.AsyncAllCollect(ctx, items, processItem)
	if err != nil {
		fmt.Printf("AsyncAllCollect encountered errors: %v\n", err)
		// We can still use partial results
	}

	fmt.Println("AsyncAllCollect results:", results)
}

// AsyncPool Example
func asyncPoolExample(ctx context.Context) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	concurrency := 3

	results, err := asyncx.AsyncPool(ctx, items, concurrency, processItem)
	if err != nil {
		log.Printf("AsyncPool failed: %v", err)
		return
	}

	fmt.Println("AsyncPool results:", results)
}

// AsyncPoolCollect Example
func asyncPoolCollectExample(ctx context.Context) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	concurrency := 3

	results, err := asyncx.AsyncPoolCollect(ctx, items, concurrency, processItem)
	if err != nil {
		fmt.Printf("AsyncPoolCollect encountered errors: %v\n", err)
		// We can still use partial results
	}

	fmt.Println("AsyncPoolCollect results:", results)
}

// Example batch processor
func processBatch(ctx context.Context, batch []int) ([]string, error) {
	fmt.Printf("Processing batch of size %d\n", len(batch))

	// Process each item in the batch
	results := make([]string, len(batch))
	for i, item := range batch {
		// Simulate batch processing
		select {
		case <-time.After(100 * time.Millisecond):
			// 5% chance of batch failure
			if rand.Intn(20) == 0 {
				return nil, fmt.Errorf("batch processing failed at item %d", item)
			}
			results[i] = fmt.Sprintf("Batch-Processed-%d", item)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return results, nil
}

// AsyncBatch Example
func asyncBatchExample(ctx context.Context) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	batchSize := 4

	results, err := asyncx.AsyncBatch(ctx, items, batchSize, processBatch)
	if err != nil {
		log.Printf("AsyncBatch failed: %v", err)
		return
	}

	fmt.Println("AsyncBatch results:", results)
}

// AsyncBatchCollect Example
func asyncBatchCollectExample(ctx context.Context) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	batchSize := 4

	results, err := asyncx.AsyncBatchCollect(ctx, items, batchSize, processBatch)
	if err != nil {
		fmt.Printf("AsyncBatchCollect encountered errors: %v\n", err)
		// We can still use partial results
	}

	fmt.Println("AsyncBatchCollect results:", results)
}

// Map function example
func doubleValue(ctx context.Context, item int) (int, error) {
	// Simulate processing time
	select {
	case <-time.After(50 * time.Millisecond):
		if item == 3 { // Simulate error for specific value
			return 0, fmt.Errorf("cannot process item 3")
		}
		return item * 2, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// Map Example
func mapExample(ctx context.Context) {
	items := []int{1, 2, 3, 4, 5}
	concurrency := 3

	results, err := asyncx.Map(ctx, items, concurrency, doubleValue)
	if err != nil {
		log.Printf("Map failed: %v", err)
		return
	}

	fmt.Println("Map results:", results)
}

// MapCollect Example
func mapCollectExample(ctx context.Context) {
	items := []int{1, 2, 3, 4, 5}
	concurrency := 3

	results, err := asyncx.MapCollect(ctx, items, concurrency, doubleValue)
	if err != nil {
		fmt.Printf("MapCollect encountered errors: %v\n", err)
		// We can still use partial results
	}

	fmt.Println("MapCollect results:", results)
}

// Filter function example
func isEvenNumberAsync(ctx context.Context, item int) (bool, error) {
	// Simulate processing time
	select {
	case <-time.After(50 * time.Millisecond):
		if item == 3 { // Simulate error for specific value
			return false, fmt.Errorf("cannot check if 3 is even")
		}
		return item%2 == 0, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// Filter Example
func filterExample(ctx context.Context) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8}
	concurrency := 3

	results, err := asyncx.Filter(ctx, items, concurrency, isEvenNumberAsync)
	if err != nil {
		log.Printf("Filter failed: %v", err)
		return
	}

	fmt.Println("Filter results (even numbers):", results)
}

// FilterCollect Example
func filterCollectExample(ctx context.Context) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8}
	concurrency := 3

	results, errs := asyncx.FilterCollect(ctx, items, concurrency, isEvenNumberAsync)
	if errs != nil {
		fmt.Printf("FilterCollect encountered errors: %v\n", errs)
		// We can still use partial results
	}

	fmt.Println("FilterCollect results (even numbers):", results)
}

// Flaky function for retry example
func flakyFunction(ctx context.Context) (string, error) {
	// Simulate network call that might fail
	if rand.Intn(3) < 2 { // 66% chance of failure
		return "", errors.New("temporary network error")
	}
	return "success result", nil
}

// RetryWithBackoff Example
func retryWithBackoffExample(ctx context.Context) {
	result, err := asyncx.RetryWithBackoff(ctx, 5, 100*time.Millisecond, flakyFunction)
	if err != nil {
		log.Printf("RetryWithBackoff failed after all attempts: %v", err)
		return
	}

	fmt.Println("RetryWithBackoff result:", result)
}

// Example functions for FirstSuccess
func createProviders() []func(context.Context) (string, error) {
	return []func(context.Context) (string, error){
		// Fast but unreliable provider
		func(ctx context.Context) (string, error) {
			time.Sleep(50 * time.Millisecond)
			if rand.Intn(10) < 8 { // 80% failure rate
				return "", errors.New("fast provider failed")
			}
			return "result from fast provider", nil
		},
		// Medium speed, medium reliability
		func(ctx context.Context) (string, error) {
			time.Sleep(150 * time.Millisecond)
			if rand.Intn(10) < 5 { // 50% failure rate
				return "", errors.New("medium provider failed")
			}
			return "result from medium provider", nil
		},
		// Slow but reliable provider
		func(ctx context.Context) (string, error) {
			time.Sleep(300 * time.Millisecond)
			if rand.Intn(10) < 2 { // 20% failure rate
				return "", errors.New("slow provider failed")
			}
			return "result from slow provider", nil
		},
	}
}

// FirstSuccess Example
func firstSuccessExample(ctx context.Context) {
	providers := createProviders()

	result, err := asyncx.FirstSuccess(ctx, providers)
	if err != nil {
		log.Printf("FirstSuccess failed: %v", err)
		return
	}

	fmt.Println("FirstSuccess result:", result)
}

// Real-world example: Concurrent API fetching
func realWorldExample(ctx context.Context) {
	// Simulate fetching data from multiple API endpoints
	urls := []string{
		"https://api.example.com/users",
		"https://api.example.com/products",
		"https://api.example.com/orders",
		"https://api.example.com/inventory",
	}

	fetchURL := func(ctx context.Context, url string) (string, error) {
		_, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return "", err
		}

		// In a real application, you would use a real client
		// client := &http.Client{}
		// resp, err := client.Do(req)

		// Simulate API response
		time.Sleep(200 * time.Millisecond)

		// Simulate occasional failures
		if rand.Intn(10) == 0 {
			return "", fmt.Errorf("failed to fetch %s", url)
		}

		return "Data from " + url, nil
	}

	// Use AsyncAll to fetch all endpoints concurrently
	results, err := asyncx.AsyncAll(ctx, urls, fetchURL)
	if err != nil {
		log.Printf("API fetching failed: %v", err)
		return
	}

	for i, data := range results {
		fmt.Printf("API %s: %s\n", urls[i], data)
	}

	// Alternative: Use AsyncPoolCollect to limit concurrency and collect all errors
	concurrentResults, errs := asyncx.AsyncPoolCollect(ctx, urls, 2, fetchURL)
	if errs != nil {
		fmt.Printf("Some API calls failed: %v\n", errs)
		// We can still process successful results
	}

	for i, data := range concurrentResults {
		if data != "" {
			fmt.Printf("Concurrent API %s: %s\n", urls[i], data)
		}
	}
}

