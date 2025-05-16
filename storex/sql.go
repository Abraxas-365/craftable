package storex

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

// PgRepository is a PostgreSQL implementation of Repository
type PgRepository[T any] struct {
	db        *sqlx.DB
	tableName string
	idField   string
}

// NewPgRepository creates a new PostgreSQL repository
func NewPgRepository[T any](db *sqlx.DB, tableName string, idField string) *PgRepository[T] {
	if idField == "" {
		idField = "id"
	}
	return &PgRepository[T]{
		db:        db,
		tableName: tableName,
		idField:   idField,
	}
}

// Create adds a new entity to the database
func (r *PgRepository[T]) Create(ctx context.Context, item T) (T, error) {
	var empty T
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	fields := []string{}
	placeholders := []string{}
	values := []interface{}{}

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}

		// Skip the ID field if it's empty
		if tag == r.idField && isEmptyValue(v.Field(i)) {
			continue
		}

		fields = append(fields, tag)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(values)+1))
		values = append(values, v.Field(i).Interface())
	}

	if len(fields) == 0 {
		return empty, storeErrors.NewWithMessage(ErrInvalidQuery, "No fields to insert")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		r.tableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)

	var result T
	err := r.db.GetContext(ctx, &result, query, values...)
	if err != nil {
		return empty, storeErrors.NewWithCause(ErrCreateFailed, err)
	}

	return result, nil
}

// FindByID retrieves an entity by its ID
func (r *PgRepository[T]) FindByID(ctx context.Context, id string) (T, error) {
	var result T
	var empty T

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", r.tableName, r.idField)
	err := r.db.GetContext(ctx, &result, query, id)

	if err != nil {
		if err == sql.ErrNoRows {
			return empty, storeErrors.NewWithMessage(ErrRecordNotFound, "ID: "+id)
		}
		return empty, storeErrors.NewWithCause(ErrSQLQueryFailed, err)
	}

	return result, nil
}

// FindOne retrieves a single entity that matches the filter
func (r *PgRepository[T]) FindOne(ctx context.Context, filter map[string]any) (T, error) {
	var result T
	var empty T

	if len(filter) == 0 {
		return empty, storeErrors.NewWithMessage(ErrInvalidQuery, "No filter provided")
	}

	conditions := []string{}
	values := []interface{}{}
	i := 1

	for k, v := range filter {
		conditions = append(conditions, fmt.Sprintf("%s = $%d", k, i))
		values = append(values, v)
		i++
	}

	whereClause := strings.Join(conditions, " AND ")
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT 1", r.tableName, whereClause)

	err := r.db.GetContext(ctx, &result, query, values...)
	if err != nil {
		if err == sql.ErrNoRows {
			return empty, storeErrors.NewWithMessage(ErrRecordNotFound, "Filter: "+whereClause)
		}
		return empty, storeErrors.NewWithCause(ErrSQLQueryFailed, err)
	}

	return result, nil
}

// Update modifies an existing entity
func (r *PgRepository[T]) Update(ctx context.Context, id string, item T) (T, error) {
	var empty T
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	setClause := []string{}
	values := []interface{}{}
	i := 1

	for j := 0; j < v.NumField(); j++ {
		field := t.Field(j)
		tag := field.Tag.Get("db")
		if tag == "" || tag == "-" || tag == r.idField {
			continue
		}

		setClause = append(setClause, fmt.Sprintf("%s = $%d", tag, i))
		values = append(values, v.Field(j).Interface())
		i++
	}

	if len(setClause) == 0 {
		return empty, storeErrors.NewWithMessage(ErrInvalidQuery, "No fields to update")
	}

	values = append(values, id)
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d RETURNING *",
		r.tableName,
		strings.Join(setClause, ", "),
		r.idField,
		i,
	)

	var result T
	err := r.db.GetContext(ctx, &result, query, values...)
	if err != nil {
		if err == sql.ErrNoRows {
			return empty, storeErrors.NewWithMessage(ErrRecordNotFound, "ID: "+id)
		}
		return empty, storeErrors.NewWithCause(ErrUpdateFailed, err)
	}

	return result, nil
}

// Delete removes an entity from the store
func (r *PgRepository[T]) Delete(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", r.tableName, r.idField)
	result, err := r.db.ExecContext(ctx, query, id)

	if err != nil {
		return storeErrors.NewWithCause(ErrDeleteFailed, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return storeErrors.NewWithCause(ErrSQLExecFailed, err)
	}

	if rowsAffected == 0 {
		return storeErrors.NewWithMessage(ErrRecordNotFound, "ID: "+id)
	}

	return nil
}

// Paginate retrieves entities with pagination
func (r *PgRepository[T]) Paginate(ctx context.Context, opts PaginationOptions) (Paginated[T], error) {
	// Process fields selection
	fieldsClause := "*"
	if len(opts.Fields) > 0 {
		fieldsClause = strings.Join(opts.Fields, ", ")
	}

	// Process filters
	whereClause := ""
	params := []interface{}{}

	if len(opts.Filters) > 0 {
		conditions := []string{}
		i := 1

		for k, v := range opts.Filters {
			conditions = append(conditions, fmt.Sprintf("%s = $%d", k, i))
			params = append(params, v)
			i++
		}

		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Process ordering
	orderClause := ""
	if opts.OrderBy != "" {
		direction := "ASC"
		if opts.Desc {
			direction = "DESC"
		}
		orderClause = fmt.Sprintf(" ORDER BY %s %s", opts.OrderBy, direction)
	}

	// Calculate pagination
	offset := (opts.Page - 1) * opts.PageSize
	limitOffset := fmt.Sprintf(" LIMIT %d OFFSET %d", opts.PageSize, offset)

	// Build queries
	dataQuery := fmt.Sprintf(
		"SELECT %s FROM %s%s%s%s",
		fieldsClause, r.tableName, whereClause, orderClause, limitOffset,
	)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", r.tableName, whereClause)

	// Execute queries
	var items []T
	var total int

	err := r.db.SelectContext(ctx, &items, dataQuery, params...)
	if err != nil {
		return Paginated[T]{}, storeErrors.NewWithCause(ErrSQLQueryFailed, err)
	}

	err = r.db.GetContext(ctx, &total, countQuery, params...)
	if err != nil {
		return Paginated[T]{}, storeErrors.NewWithCause(ErrSQLCountFailed, err)
	}

	return NewPaginated(items, opts.Page, opts.PageSize, total), nil
}

// PgBulkOperator implements BulkOperator for PostgreSQL
type PgBulkOperator[T any] struct {
	*PgRepository[T]
}

// NewPgBulkOperator creates a new PostgreSQL bulk operator
func NewPgBulkOperator[T any](repo *PgRepository[T]) *PgBulkOperator[T] {
	return &PgBulkOperator[T]{repo}
}

// BulkInsert adds multiple entities in a single operation
func (b *PgBulkOperator[T]) BulkInsert(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	// Get the structure of the first item to determine fields
	v := reflect.ValueOf(items[0])
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	fields := []string{}

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}

		// Skip ID field if it's empty
		if tag == b.idField && isEmptyValue(v.Field(i)) {
			continue
		}

		fields = append(fields, tag)
	}

	if len(fields) == 0 {
		return storeErrors.NewWithMessage(ErrInvalidQuery, "No fields to insert")
	}

	// Build query with multiple value groups
	valueGroups := []string{}
	valueParams := []interface{}{}
	paramIndex := 1

	for _, item := range items {
		v := reflect.ValueOf(item)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		placeholders := []string{}

		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("db")
			if tag == "" || tag == "-" {
				continue
			}

			// Skip ID field if it's empty
			if tag == b.idField && isEmptyValue(v.Field(i)) {
				continue
			}

			// Check if this field is in our fields list
			found := false
			for _, f := range fields {
				if f == tag {
					found = true
					break
				}
			}

			if found {
				placeholders = append(placeholders, fmt.Sprintf("$%d", paramIndex))
				valueParams = append(valueParams, v.Field(i).Interface())
				paramIndex++
			}
		}

		valueGroups = append(valueGroups, "("+strings.Join(placeholders, ", ")+")")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		b.tableName,
		strings.Join(fields, ", "),
		strings.Join(valueGroups, ", "),
	)

	_, err := b.db.ExecContext(ctx, query, valueParams...)
	if err != nil {
		return storeErrors.NewWithCause(ErrBulkOpFailed, err)
	}

	return nil
}

// BulkUpdate modifies multiple entities in a single operation
func (b *PgBulkOperator[T]) BulkUpdate(ctx context.Context, items []T) error {
	// Using transactions for bulk updates
	tx, err := b.db.BeginTxx(ctx, nil)
	if err != nil {
		return storeErrors.NewWithCause(ErrTxBeginFailed, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, item := range items {
		v := reflect.ValueOf(item)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		// Find the ID field
		t := v.Type()
		var id interface{}
		var found bool

		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("db")
			if tag == b.idField {
				id = v.Field(i).Interface()
				found = true
				break
			}
		}

		if !found || id == nil {
			return storeErrors.NewWithMessage(ErrInvalidID, "Missing ID for bulk update")
		}

		// Build update for this item
		setClause := []string{}
		values := []interface{}{}
		paramIndex := 1

		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("db")
			if tag == "" || tag == "-" || tag == b.idField {
				continue
			}

			setClause = append(setClause, fmt.Sprintf("%s = $%d", tag, paramIndex))
			values = append(values, v.Field(i).Interface())
			paramIndex++
		}

		if len(setClause) == 0 {
			continue // Nothing to update
		}

		values = append(values, id)
		query := fmt.Sprintf(
			"UPDATE %s SET %s WHERE %s = $%d",
			b.tableName,
			strings.Join(setClause, ", "),
			b.idField,
			paramIndex,
		)

		_, err = tx.ExecContext(ctx, query, values...)
		if err != nil {
			return storeErrors.NewWithCause(ErrUpdateFailed, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return storeErrors.NewWithCause(ErrTxCommitFailed, err)
	}

	return nil
}

// BulkDelete removes multiple entities in a single operation
func (b *PgBulkOperator[T]) BulkDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	params := make([]interface{}, len(ids))

	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		params[i] = id
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s IN (%s)",
		b.tableName,
		b.idField,
		strings.Join(placeholders, ", "),
	)

	result, err := b.db.ExecContext(ctx, query, params...)
	if err != nil {
		return storeErrors.NewWithCause(ErrBulkOpFailed, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return storeErrors.NewWithCause(ErrSQLExecFailed, err)
	}

	if rowsAffected == 0 {
		return storeErrors.NewWithMessage(ErrRecordNotFound, "No records matched IDs for deletion")
	}

	return nil
}

// PgTxManager provides transaction support for PostgreSQL
type PgTxManager struct {
	db *sqlx.DB
}

// NewPgTxManager creates a new PostgreSQL transaction manager
func NewPgTxManager(db *sqlx.DB) *PgTxManager {
	return &PgTxManager{db: db}
}

// WithTransaction executes operations within a transaction
func (tm *PgTxManager) WithTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	tx, err := tm.db.BeginTxx(ctx, nil)
	if err != nil {
		return storeErrors.NewWithCause(ErrTxBeginFailed, err)
	}

	txCtx := context.WithValue(ctx, "tx", tx)

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // Re-throw panic after rollback
		} else if err != nil {
			_ = tx.Rollback()
		}
	}()

	err = fn(txCtx)
	if err != nil {
		return err // Error is already wrapped by the function
	}

	if err = tx.Commit(); err != nil {
		return storeErrors.NewWithCause(ErrTxCommitFailed, err)
	}

	return nil
}

// PgSearchable provides full-text search for PostgreSQL
type PgSearchable[T any] struct {
	*PgRepository[T]
}

// NewPgSearchable creates a new PostgreSQL searchable repository
func NewPgSearchable[T any](repo *PgRepository[T]) *PgSearchable[T] {
	return &PgSearchable[T]{repo}
}

// Search performs a full-text search using PostgreSQL's ts_vector
func (s *PgSearchable[T]) Search(ctx context.Context, query string, opts SearchOptions) ([]T, error) {
	if len(opts.Fields) == 0 {
		return nil, storeErrors.NewWithMessage(ErrInvalidQuery, "No search fields specified")
	}

	// Default values
	if opts.Limit <= 0 {
		opts.Limit = 25
	}

	// Build the search query using to_tsquery
	searchQuery := fmt.Sprintf("to_tsquery('english', '%s')",
		strings.ReplaceAll(query, " ", " & "))

	// Build the WHERE clause with rank ordering
	tsVectorExpressions := []string{}
	rankExpressions := []string{}

	for _, field := range opts.Fields {
		boost := 1.0
		if weight, ok := opts.Boost[field]; ok {
			boost = weight
		}

		tsVectorExpressions = append(tsVectorExpressions,
			fmt.Sprintf("to_tsvector('english', %s)", field))

		rankExpressions = append(rankExpressions,
			fmt.Sprintf("ts_rank(to_tsvector('english', %s), %s) * %f",
				field, searchQuery, boost))
	}

	whereClause := fmt.Sprintf("(%s) @@ %s",
		strings.Join(tsVectorExpressions, " || "), searchQuery)

	rankClause := "(" + strings.Join(rankExpressions, " + ") + ") AS search_rank"

	// Build the complete query
	sqlQuery := fmt.Sprintf(
		"SELECT *, %s FROM %s WHERE %s ORDER BY search_rank DESC LIMIT %d OFFSET %d",
		rankClause, s.tableName, whereClause, opts.Limit, opts.Offset,
	)

	var results []T
	err := s.db.SelectContext(ctx, &results, sqlQuery)
	if err != nil {
		return nil, storeErrors.NewWithCause(ErrSearchFailed, err)
	}

	return results, nil
}

// PgChangeStream provides real-time notifications for PostgreSQL using LISTEN/NOTIFY
type PgChangeStream[T any] struct {
	*PgRepository[T]
	listener *pq.Listener
	channel  string
}

// NewPgChangeStream creates a new PostgreSQL change stream
// Requires appropriate triggers to be set up on the table
func NewPgChangeStream[T any](repo *PgRepository[T], connStr string, channel string) (*PgChangeStream[T], error) {
	if channel == "" {
		channel = repo.tableName + "_changes"
	}

	listener := pq.NewListener(connStr, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			fmt.Printf("Error in listener: %v\n", err)
		}
	})

	err := listener.Listen(channel)
	if err != nil {
		return nil, storeErrors.NewWithCause(ErrConnectionFailed, err)
	}

	return &PgChangeStream[T]{
		PgRepository: repo,
		listener:     listener,
		channel:      channel,
	}, nil
}

// Watch creates a stream of change events
func (cs *PgChangeStream[T]) Watch(ctx context.Context, filter map[string]any) (<-chan ChangeEvent[T], error) {
	events := make(chan ChangeEvent[T])

	go func() {
		defer close(events)

		for {
			select {
			case <-ctx.Done():
				return

			case n := <-cs.listener.Notify:
				if n == nil {
					continue
				}

				// Parse the notification payload (JSON expected)
				var payload struct {
					Operation string    `json:"operation"`
					ID        string    `json:"id"`
					Timestamp time.Time `json:"timestamp"`
				}

				err := json.Unmarshal([]byte(n.Extra), &payload)
				if err != nil {
					continue
				}

				// Apply filter if provided
				if len(filter) > 0 {
					match := true
					for k, v := range filter {
						if k == "operation" && payload.Operation != v.(string) {
							match = false
							break
						}
					}

					if !match {
						continue
					}
				}

				// Get old/new values
				var oldValue, newValue *T

				if payload.Operation != "delete" {
					item, err := cs.FindByID(ctx, payload.ID)
					if err == nil {
						newValue = &item
					}
				}

				event := ChangeEvent[T]{
					Operation: payload.Operation,
					OldValue:  oldValue,
					NewValue:  newValue,
					Timestamp: payload.Timestamp,
				}

				events <- event
			}
		}
	}()

	return events, nil
}

// Helper function to check if a value is empty
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

