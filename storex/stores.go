package storex

import (
	"context"
	"reflect"
	"strings"
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

// Repository defines the basic CRUD operations for any entity
type Repository[T any] interface {
	// Create adds a new entity to the store
	Create(ctx context.Context, item T) (T, error)

	// FindByID retrieves an entity by its ID
	FindByID(ctx context.Context, id string) (T, error)

	// FindOne retrieves a single entity that matches the filter
	FindOne(ctx context.Context, filter map[string]any) (T, error)

	// Update modifies an existing entity
	Update(ctx context.Context, id string, item T) (T, error)

	// Delete removes an entity from the store
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

// QueryBuilder provides a type-safe way to construct queries
type QueryBuilder[T any] struct {
	filters []Filter
	sorts   []Sort
	limit   int
	offset  int
	fields  []string
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

// Searchable provides full-text search capabilities
type Searchable[T any] interface {
	// Search performs a full-text search
	Search(ctx context.Context, query string, opts SearchOptions) ([]T, error)
}

// SearchOptions configures a search operation
type SearchOptions struct {
	Fields []string           // Fields to search in
	Boost  map[string]float64 // Field boosting factors
	Limit  int                // Maximum results to return
	Offset int                // Number of results to skip
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

func extractIDValue(item interface{}, idField string) interface{} {
	// Get the value of the item
	val := reflect.ValueOf(item)

	// If item is a pointer, get the value it points to
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	// Make sure we're dealing with a struct
	if val.Kind() != reflect.Struct {
		return nil
	}

	// Get the type for field lookup
	typ := val.Type()

	// First, try to find the field directly by name
	field := val.FieldByName(idField)
	if field.IsValid() && field.CanInterface() {
		return field.Interface()
	}

	// If not found, try to match by DB tag
	for i := 0; i < typ.NumField(); i++ {
		fieldType := typ.Field(i)

		// Check if the field has the db tag matching the idField
		dbTag := fieldType.Tag.Get("db")
		if dbTag == idField {
			// Found a match with the tag
			field = val.Field(i)
			if field.IsValid() && field.CanInterface() {
				return field.Interface()
			}
		}

		// Also check if the lowercase field name matches (convention)
		if strings.EqualFold(fieldType.Name, idField) {
			field = val.Field(i)
			if field.IsValid() && field.CanInterface() {
				return field.Interface()
			}
		}
	}

	// If we couldn't find a match, look for fields with common ID names
	if strings.EqualFold(idField, "id") {
		commonIDFields := []string{"ID", "Id", "Uuid", "UUID", "Guid", "GUID"}
		for _, name := range commonIDFields {
			field := val.FieldByName(name)
			if field.IsValid() && field.CanInterface() {
				return field.Interface()
			}
		}
	}

	// Couldn't find a matching field
	return nil
}
