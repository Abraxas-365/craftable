package storexinmemory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Abraxas-365/craftable/storex"
)

// MemoryStore is an in-memory implementation of Repository, BulkOperator, TxManager, and ChangeStream
type MemoryStore[T any] struct {
	data              map[string]T
	mu                sync.RWMutex
	idExtractor       func(T) string
	changeSubscribers []chan storex.ChangeEvent[T]
	idGenerator       func() string
}

// MemoryStoreOption defines a functional option for configuring MemoryStore
type MemoryStoreOption[T any] func(*MemoryStore[T])

// WithIDExtractor configures the function that extracts an ID from an entity
func WithIDExtractor[T any](extractor func(T) string) MemoryStoreOption[T] {
	return func(ms *MemoryStore[T]) {
		ms.idExtractor = extractor
	}
}

// WithIDGenerator configures the function that generates new IDs
func WithIDGenerator[T any](generator func() string) MemoryStoreOption[T] {
	return func(ms *MemoryStore[T]) {
		ms.idGenerator = generator
	}
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore[T any](options ...MemoryStoreOption[T]) *MemoryStore[T] {
	ms := &MemoryStore[T]{
		data: make(map[string]T),
		// Default ID generator (should be overridden for production use)
		idGenerator: func() string {
			return fmt.Sprintf("%d", time.Now().UnixNano())
		},
	}

	// Apply options
	for _, option := range options {
		option(ms)
	}

	return ms
}

// Create adds a new entity to the store
func (ms *MemoryStore[T]) Create(ctx context.Context, item T) (T, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// If the item has no ID, generate one
	id := ""
	if ms.idExtractor != nil {
		id = ms.idExtractor(item)
	}

	if id == "" {
		// This requires the ability to set the ID back to the item,
		// which may not be possible with a generic T
		// We'd need a more specific approach for this
		id = ms.idGenerator()
	}

	// Check if record already exists
	if _, exists := ms.data[id]; exists {
		var zero T
		return zero, storex.StoreErrors.New(storex.ErrCreateFailed).WithDetail("reason", "ID already exists")
	}

	ms.data[id] = item

	// Notify subscribers about the change
	ms.notifyChange("insert", nil, &item)

	return item, nil
}

// FindByID retrieves an entity by its ID
func (ms *MemoryStore[T]) FindByID(ctx context.Context, id string) (T, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if item, exists := ms.data[id]; exists {
		return item, nil
	}

	var zero T
	return zero, storex.StoreErrors.New(storex.ErrRecordNotFound).WithDetail("id", id)
}

// FindOne retrieves a single entity that matches the filter
func (ms *MemoryStore[T]) FindOne(ctx context.Context, filter map[string]any) (T, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for _, item := range ms.data {
		if ms.matchesFilter(item, filter) {
			return item, nil
		}
	}

	var zero T
	return zero, storex.StoreErrors.New(storex.ErrRecordNotFound).WithDetail("filter", filter)
}

// matchesFilter checks if an item matches the given filter
func (ms *MemoryStore[T]) matchesFilter(item T, filter map[string]any) bool {
	// This is a simplified implementation that would need to be expanded
	// based on the actual structure of T and how filters should work
	// In a real implementation, you'd use reflection or another mechanism
	// to check if the item's fields match the filter criteria
	return true
}

// Update modifies an existing entity
func (ms *MemoryStore[T]) Update(ctx context.Context, id string, item T) (T, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	oldItem, exists := ms.data[id]
	if !exists {
		var zero T
		return zero, storex.StoreErrors.New(storex.ErrRecordNotFound).WithDetail("id", id)
	}

	ms.data[id] = item

	// Notify subscribers about the change
	ms.notifyChange("update", &oldItem, &item)

	return item, nil
}

// Delete removes an entity from the store
func (ms *MemoryStore[T]) Delete(ctx context.Context, id string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	item, exists := ms.data[id]
	if !exists {
		return storex.StoreErrors.New(storex.ErrRecordNotFound).WithDetail("id", id)
	}

	delete(ms.data, id)

	// Notify subscribers about the change
	ms.notifyChange("delete", &item, nil)

	return nil
}

// Paginate retrieves entities with pagination
func (ms *MemoryStore[T]) Paginate(ctx context.Context, opts storex.PaginationOptions) (storex.Paginated[T], error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Filter items
	var filteredItems []T
	for _, item := range ms.data {
		if ms.matchesFilter(item, opts.Filters) {
			filteredItems = append(filteredItems, item)
		}
	}

	// Sort items (simplified - would need to be expanded for real use)
	// In a real implementation, you'd use reflection or another mechanism
	// to sort based on the orderBy field

	// Apply pagination
	total := len(filteredItems)
	start := (opts.Page - 1) * opts.PageSize
	end := start + opts.PageSize

	if start >= total {
		return storex.NewPaginated([]T{}, opts.Page, opts.PageSize, total), nil
	}

	if end > total {
		end = total
	}

	return storex.NewPaginated(filteredItems[start:end], opts.Page, opts.PageSize, total), nil
}

// BulkInsert adds multiple entities in a single operation
func (ms *MemoryStore[T]) BulkInsert(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, item := range items {
		id := ""
		if ms.idExtractor != nil {
			id = ms.idExtractor(item)
		}

		if id == "" {
			id = ms.idGenerator()
		}

		if _, exists := ms.data[id]; exists {
			return storex.StoreErrors.New(storex.ErrBulkOpFailed).WithDetail("reason", "duplicate ID found")
		}

		ms.data[id] = item
		ms.notifyChange("insert", nil, &item)
	}

	return nil
}

// BulkUpdate modifies multiple entities in a single operation
func (ms *MemoryStore[T]) BulkUpdate(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, item := range items {
		id := ""
		if ms.idExtractor != nil {
			id = ms.idExtractor(item)
		}

		if id == "" {
			return storex.StoreErrors.New(storex.ErrBulkOpFailed).WithDetail("reason", "item has no ID")
		}

		oldItem, exists := ms.data[id]
		if !exists {
			return storex.StoreErrors.New(storex.ErrBulkOpFailed).WithDetail("reason", "item not found")
		}

		ms.data[id] = item
		ms.notifyChange("update", &oldItem, &item)
	}

	return nil
}

// BulkDelete removes multiple entities in a single operation
func (ms *MemoryStore[T]) BulkDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, id := range ids {
		item, exists := ms.data[id]
		if !exists {
			return storex.StoreErrors.New(storex.ErrBulkOpFailed).WithDetail("item not found id", id)
		}

		delete(ms.data, id)
		ms.notifyChange("delete", &item, nil)
	}

	return nil
}

// WithTransaction executes operations within a transaction
// For the in-memory implementation, we just run the function directly
// since there's no actual transaction support needed
func (ms *MemoryStore[T]) WithTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	// In a real implementation with a database, this would begin a transaction,
	// call fn with the transaction context, and then commit or rollback.
	// For in-memory, we just call the function directly.
	return fn(ctx)
}

// Search performs a full-text search
func (ms *MemoryStore[T]) Search(ctx context.Context, query string, opts storex.SearchOptions) ([]T, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// This is a simplified implementation. In a real search implementation,
	// you'd implement a more sophisticated text search algorithm
	var results []T
	for _, item := range ms.data {
		// In a real implementation, you'd check if the item matches the search query
		// For now, we just return all items
		results = append(results, item)
	}

	// Apply limit and offset
	if opts.Limit > 0 && opts.Offset >= 0 {
		end := opts.Offset + opts.Limit
		if end > len(results) {
			end = len(results)
		}

		if opts.Offset < len(results) {
			results = results[opts.Offset:end]
		} else {
			results = []T{}
		}
	}

	return results, nil
}

// Watch creates a stream of change events
func (ms *MemoryStore[T]) Watch(ctx context.Context, filter map[string]any) (<-chan storex.ChangeEvent[T], error) {
	ms.mu.Lock()
	ch := make(chan storex.ChangeEvent[T], 100) // Buffered channel
	ms.changeSubscribers = append(ms.changeSubscribers, ch)
	ms.mu.Unlock()

	// Close the channel when the context is done
	go func() {
		<-ctx.Done()
		ms.mu.Lock()
		for i, subscriber := range ms.changeSubscribers {
			if subscriber == ch {
				// Remove this subscriber
				ms.changeSubscribers = append(ms.changeSubscribers[:i], ms.changeSubscribers[i+1:]...)
				break
			}
		}
		ms.mu.Unlock()
		close(ch)
	}()

	return ch, nil
}

// notifyChange sends change notifications to subscribers
func (ms *MemoryStore[T]) notifyChange(operation string, oldValue, newValue *T) {
	event := storex.ChangeEvent[T]{
		Operation: operation,
		OldValue:  oldValue,
		NewValue:  newValue,
		Timestamp: time.Now(),
	}

	// Send to all subscribers without blocking
	for _, ch := range ms.changeSubscribers {
		select {
		case ch <- event:
			// Sent successfully
		default:
			// Channel buffer is full, skip this notification
		}
	}
}

// Count returns the total number of items in the store
func (ms *MemoryStore[T]) Count(ctx context.Context, filter map[string]any) (int, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if filter == nil || len(filter) == 0 {
		return len(ms.data), nil
	}

	count := 0
	for _, item := range ms.data {
		if ms.matchesFilter(item, filter) {
			count++
		}
	}

	return count, nil
}

// Clear removes all items from the store
func (ms *MemoryStore[T]) Clear() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.data = make(map[string]T)
}
