package storex

import (
	"context"
	"time"
)

// Page represents pagination metadata
type Page struct {
	Number int `json:"page"`      // Current page number (1-based)
	Size   int `json:"page_size"` // Number of records per page
	Total  int `json:"total"`     // Total number of records
	Pages  int `json:"pages"`     // Total number of pages
}

// Paginated is a generic container for paginated data with metadata
type Paginated[T any] struct {
	Data  []T  `json:"data"`       // The paginated items
	Page  Page `json:"pagination"` // Pagination metadata
	Empty bool `json:"empty"`      // Whether the result contains any items
}

// NewPaginated creates a new paginated result with calculated fields
func NewPaginated[T any](data []T, page, size, total int) Paginated[T] {
	pages := 0
	if size > 0 {
		pages = (total + size - 1) / size // Ceiling division
	}

	return Paginated[T]{
		Data: data,
		Page: Page{
			Number: page,
			Size:   size,
			Total:  total,
			Pages:  pages,
		},
		Empty: len(data) == 0,
	}
}

// HasNext returns whether there are more pages after the current one
func (p Paginated[T]) HasNext() bool {
	return p.Page.Number < p.Page.Pages
}

// HasPrevious returns whether there are pages before the current one
func (p Paginated[T]) HasPrevious() bool {
	return p.Page.Number > 1
}

// PaginationOptions holds options for pagination queries
type PaginationOptions struct {
	Page     int            // Page number (1-based)
	PageSize int            // Number of records per page
	OrderBy  string         // Field to order by (format depends on database)
	Desc     bool           // Whether to sort in descending order
	Filters  map[string]any // Optional filters
	Fields   []string       // Optional field selection
}

// DefaultPaginationOptions returns sensible default options
func DefaultPaginationOptions() PaginationOptions {
	return PaginationOptions{
		Page:     1,
		PageSize: 25,
		OrderBy:  "id",
		Desc:     false,
		Filters:  make(map[string]any),
		Fields:   nil,
	}
}

// WithFilter adds a filter to the pagination options
func (o PaginationOptions) WithFilter(key string, value any) PaginationOptions {
	if o.Filters == nil {
		o.Filters = make(map[string]any)
	}
	o.Filters[key] = value
	return o
}

// Repository provides a generic data access interface for entity operations.
// The interface uses string-based IDs for flexibility across different ID types.
type Repository[T any] interface {
	// Create adds a new entity to the store
	Create(ctx context.Context, item T) (T, error)

	// FindByID retrieves an entity by its ID
	//
	// Usage with integer IDs:
	//   entity, err := repo.FindByID(ctx, strconv.Itoa(42))
	//
	// Usage with UUID IDs:
	//   entity, err := repo.FindByID(ctx, myUUID.String())
	FindByID(ctx context.Context, id string) (T, error)

	// FindOne retrieves a single entity that matches the filter
	FindOne(ctx context.Context, filter map[string]any) (T, error)

	// Update modifies an existing entity
	//
	// Usage with integer IDs:
	//   updatedEntity, err := repo.Update(ctx, strconv.Itoa(42), entityToUpdate)
	//
	// Usage with UUID IDs:
	//   updatedEntity, err := repo.Update(ctx, myUUID.String(), entityToUpdate)
	Update(ctx context.Context, id string, item T) (T, error)

	// Delete removes an entity from the store
	//
	// Usage with integer IDs:
	//   err := repo.Delete(ctx, strconv.Itoa(42))
	//
	// Usage with UUID IDs:
	//   err := repo.Delete(ctx, myUUID.String())
	Delete(ctx context.Context, id string) error

	// Paginate retrieves entities with pagination
	Paginate(ctx context.Context, opts PaginationOptions) (Paginated[T], error)
}

// BulkOperator provides batch operations for efficiency
type BulkOperator[T any] interface {
	// BulkInsert adds multiple entities in a single operation
	BulkInsert(ctx context.Context, items []T) error

	// BulkUpdate modifies multiple entities in a single operation
	BulkUpdate(ctx context.Context, items []T) error

	// BulkDelete removes multiple entities in a single operation
	BulkDelete(ctx context.Context, ids []string) error
}

// TxManager provides transaction support
type TxManager interface {
	// WithTransaction executes operations within a transaction
	WithTransaction(ctx context.Context, fn func(txCtx context.Context) error) error
}

// SearchOptions configures a search operation
type SearchOptions struct {
	Fields []string           // Fields to search in
	Boost  map[string]float64 // Field boosting factors
	Limit  int                // Maximum results to return
	Offset int                // Number of results to skip
}

// Searchable provides full-text search capabilities
type Searchable[T any] interface {
	// Search performs a full-text search
	Search(ctx context.Context, query string, opts SearchOptions) ([]T, error)
}

// ChangeEvent represents a data change notification
type ChangeEvent[T any] struct {
	Operation string    // insert, update, delete
	OldValue  *T        // Previous value (nil for inserts)
	NewValue  *T        // New value (nil for deletes)
	Timestamp time.Time // When the change occurred
}

// ChangeStream provides real-time notifications for data changes
type ChangeStream[T any] interface {
	// Watch creates a stream of change events
	Watch(ctx context.Context, filter map[string]any) (<-chan ChangeEvent[T], error)
}

// QueryBuilder provides a type-safe way to construct queries
type QueryBuilder[T any] struct {
	filters []Filter
	sorts   []Sort
	limit   int
	offset  int
	fields  []string
}

// Filter represents a query filter condition
type Filter struct {
	Field string
	Op    string
	Value any
}

// Sort represents a sort order specification
type Sort struct {
	Field string
	Desc  bool
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder[T any]() *QueryBuilder[T] {
	return &QueryBuilder[T]{
		limit: -1,
	}
}

// Where adds a filter condition to the query
func (qb *QueryBuilder[T]) Where(field string, op string, value any) *QueryBuilder[T] {
	qb.filters = append(qb.filters, Filter{Field: field, Op: op, Value: value})
	return qb
}

// OrderBy adds a sort specification to the query
func (qb *QueryBuilder[T]) OrderBy(field string, desc bool) *QueryBuilder[T] {
	qb.sorts = append(qb.sorts, Sort{Field: field, Desc: desc})
	return qb
}

// Limit sets the maximum number of results to return
func (qb *QueryBuilder[T]) Limit(limit int) *QueryBuilder[T] {
	qb.limit = limit
	return qb
}

// Offset sets the number of results to skip
func (qb *QueryBuilder[T]) Offset(offset int) *QueryBuilder[T] {
	qb.offset = offset
	return qb
}

// Select specifies the fields to include in the results
func (qb *QueryBuilder[T]) Select(fields ...string) *QueryBuilder[T] {
	qb.fields = fields
	return qb
}

// ToPaginationOptions converts the query builder to pagination options
func (qb *QueryBuilder[T]) ToPaginationOptions() PaginationOptions {
	opts := DefaultPaginationOptions()

	// Convert limit/offset to page/pageSize
	if qb.limit > 0 {
		opts.PageSize = qb.limit
	}

	if qb.offset > 0 && opts.PageSize > 0 {
		opts.Page = (qb.offset / opts.PageSize) + 1
	}

	// Add filters
	for _, filter := range qb.filters {
		// Convert filter.Op to a map-based filter
		opts = opts.WithFilter(filter.Field, filter.Value)
	}

	// Add sort
	if len(qb.sorts) > 0 {
		opts.OrderBy = qb.sorts[0].Field
		opts.Desc = qb.sorts[0].Desc
	}

	// Add field selection
	opts.Fields = qb.fields

	return opts
}
