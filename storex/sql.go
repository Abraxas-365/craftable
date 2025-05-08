package storex

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// TypedSQL provides SQL operations for a specific type
type TypedSQL[T any] struct {
	DB        *sql.DB
	TableName string
	IDColumn  string
}

// NewTypedSQL creates a new TypedSQL helper for a specific type
func NewTypedSQL[T any](db *sql.DB) *TypedSQL[T] {
	return &TypedSQL[T]{
		DB:        db,
		TableName: "", // Must be set by WithTableName
		IDColumn:  "id",
	}
}

// WithTableName sets the table name for operations
func (s *TypedSQL[T]) WithTableName(tableName string) *TypedSQL[T] {
	s.TableName = tableName
	return s
}

// WithIDColumn sets the column name for the primary key
func (s *TypedSQL[T]) WithIDColumn(columnName string) *TypedSQL[T] {
	s.IDColumn = columnName
	return s
}

// Create adds a new record to the database
func (s *TypedSQL[T]) Create(ctx context.Context, item T) (T, error) {
	// Ensure table name is set
	if s.TableName == "" {
		return item, storeErrors.New(ErrInvalidQuery).
			WithDetail("reason", "table name not set")
	}

	// Extract field names and values using reflection
	fields, values, err := extractFieldsAndValues(item)
	if err != nil {
		return item, err
	}

	// Build INSERT query
	columnNames := strings.Join(fields, ", ")
	placeholders := createPlaceholders(len(fields))

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		s.TableName,
		columnNames,
		placeholders,
	)

	// Execute query with context
	row := s.DB.QueryRowContext(ctx, query, values...)

	// Scan result into item
	var result T
	err = scanIntoStruct(row, &result)
	if err != nil {
		return item, storeErrors.New(ErrSQLScanFailed).
			WithDetail("table", s.TableName).
			WithCause(err)
	}

	return result, nil
}

// FindByID retrieves a record by ID
func (s *TypedSQL[T]) FindByID(ctx context.Context, id string) (T, error) {
	var result T

	// Ensure table name is set
	if s.TableName == "" {
		return result, storeErrors.New(ErrInvalidQuery).
			WithDetail("reason", "table name not set")
	}

	// Build query
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", s.TableName, s.IDColumn)

	// Execute query
	row := s.DB.QueryRowContext(ctx, query, id)

	// Scan result
	err := scanIntoStruct(row, &result)
	if err != nil {
		if err == sql.ErrNoRows {
			return result, storeErrors.New(ErrRecordNotFound).
				WithDetail("id", id).
				WithDetail("table", s.TableName)
		}
		return result, storeErrors.New(ErrSQLScanFailed).
			WithDetail("id", id).
			WithDetail("table", s.TableName).
			WithCause(err)
	}

	return result, nil
}

// FindOne retrieves a single record matching the filter
func (s *TypedSQL[T]) FindOne(ctx context.Context, filter map[string]any) (T, error) {
	var result T

	// Ensure table name is set
	if s.TableName == "" {
		return result, storeErrors.New(ErrInvalidQuery).
			WithDetail("reason", "table name not set")
	}

	// Build WHERE clause from filter
	where, args := buildWhereClause(filter)

	// Build query
	query := fmt.Sprintf("SELECT * FROM %s %s LIMIT 1", s.TableName, where)

	// Execute query
	row := s.DB.QueryRowContext(ctx, query, args...)

	// Scan result
	err := scanIntoStruct(row, &result)
	if err != nil {
		if err == sql.ErrNoRows {
			return result, storeErrors.New(ErrRecordNotFound).
				WithDetail("filter", fmt.Sprintf("%v", filter)).
				WithDetail("table", s.TableName)
		}
		return result, storeErrors.New(ErrSQLScanFailed).
			WithDetail("filter", fmt.Sprintf("%v", filter)).
			WithDetail("table", s.TableName).
			WithCause(err)
	}

	return result, nil
}

// Update modifies an existing record
func (s *TypedSQL[T]) Update(ctx context.Context, id string, item T) (T, error) {
	var result T

	// Ensure table name is set
	if s.TableName == "" {
		return result, storeErrors.New(ErrInvalidQuery).
			WithDetail("reason", "table name not set")
	}

	// Extract fields and values using reflection
	fields, values, err := extractFieldsAndValues(item)
	if err != nil {
		return result, err
	}

	// Build SET clause
	setClause := buildSetClause(fields)

	// Add ID to values for WHERE clause
	values = append(values, id)

	// Build query
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d RETURNING *",
		s.TableName,
		setClause,
		s.IDColumn,
		len(values),
	)

	// Execute query
	row := s.DB.QueryRowContext(ctx, query, values...)

	// Scan result
	err = scanIntoStruct(row, &result)
	if err != nil {
		if err == sql.ErrNoRows {
			return result, storeErrors.New(ErrRecordNotFound).
				WithDetail("id", id).
				WithDetail("table", s.TableName)
		}
		return result, storeErrors.New(ErrSQLScanFailed).
			WithDetail("id", id).
			WithDetail("table", s.TableName).
			WithCause(err)
	}

	return result, nil
}

// Delete removes a record from the database
func (s *TypedSQL[T]) Delete(ctx context.Context, id string) error {
	// Ensure table name is set
	if s.TableName == "" {
		return storeErrors.New(ErrInvalidQuery).
			WithDetail("reason", "table name not set")
	}

	// Build query
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", s.TableName, s.IDColumn)

	// Execute query
	result, err := s.DB.ExecContext(ctx, query, id)
	if err != nil {
		return storeErrors.New(ErrSQLExecFailed).
			WithDetail("id", id).
			WithDetail("table", s.TableName).
			WithCause(err)
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return storeErrors.New(ErrSQLExecFailed).
			WithDetail("id", id).
			WithDetail("table", s.TableName).
			WithCause(err)
	}

	if rowsAffected == 0 {
		return storeErrors.New(ErrRecordNotFound).
			WithDetail("id", id).
			WithDetail("table", s.TableName)
	}

	return nil
}

// Paginate retrieves records with pagination
func (s *TypedSQL[T]) Paginate(ctx context.Context, opts PaginationOptions) (Paginated[T], error) {
	// Ensure table name is set
	if s.TableName == "" {
		return Paginated[T]{}, storeErrors.New(ErrInvalidQuery).
			WithDetail("reason", "table name not set")
	}

	// Build WHERE clause from filters
	where, args := buildWhereClause(opts.Filters)

	// Build base query
	baseQuery := fmt.Sprintf("SELECT * FROM %s %s", s.TableName, where)

	// Build count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", s.TableName, where)

	// Create scan function
	scanFn := func(rows *sql.Rows) (T, error) {
		var item T
		err := scanIntoStruct(rows, &item)
		return item, err
	}

	// Execute pagination
	return PaginateSQL(ctx, s.DB, opts, baseQuery, countQuery, args, scanFn)
}

// PaginateSimple is a non-generic method that uses the generic functions
func (s *TypedSQL[T]) PaginateSimple(
	ctx context.Context,
	opts PaginationOptions,
	query string,
	args []interface{},
) (Paginated[T], error) {
	return PaginateSimple[T](ctx, s.DB, opts, query, args)
}

// BulkInsert adds multiple records in a single operation
func (s *TypedSQL[T]) BulkInsert(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	// Ensure table name is set
	if s.TableName == "" {
		return storeErrors.New(ErrInvalidQuery).
			WithDetail("reason", "table name not set")
	}

	// Begin transaction
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return storeErrors.New(ErrTxBeginFailed).
			WithDetail("table", s.TableName).
			WithCause(err)
	}
	defer tx.Rollback()

	// Extract fields from first item (assuming all items have same structure)
	fields, _, err := extractFieldsAndValues(items[0])
	if err != nil {
		return err
	}

	// Build INSERT query
	columnNames := strings.Join(fields, ", ")

	// Prepare statement with multiple value sets
	placeholderSets := make([]string, len(items))
	args := make([]interface{}, 0, len(items)*len(fields))

	for i, item := range items {
		_, values, err := extractFieldsAndValues(item)
		if err != nil {
			return err
		}

		// Calculate placeholder indices for this item
		startIdx := i*len(fields) + 1
		placeholders := make([]string, len(fields))
		for j := range fields {
			placeholders[j] = fmt.Sprintf("$%d", startIdx+j)
		}

		placeholderSets[i] = fmt.Sprintf("(%s)", strings.Join(placeholders, ", "))
		args = append(args, values...)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		s.TableName,
		columnNames,
		strings.Join(placeholderSets, ", "),
	)

	// Execute statement
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return storeErrors.New(ErrSQLExecFailed).
			WithDetail("operation", "bulk_insert").
			WithDetail("count", len(items)).
			WithDetail("table", s.TableName).
			WithCause(err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return storeErrors.New(ErrTxCommitFailed).
			WithDetail("table", s.TableName).
			WithCause(err)
	}

	return nil
}

// BulkUpdate modifies multiple records in a single operation
func (s *TypedSQL[T]) BulkUpdate(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	// Ensure table name is set
	if s.TableName == "" {
		return storeErrors.New(ErrInvalidQuery).
			WithDetail("reason", "table name not set")
	}

	// Begin transaction
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return storeErrors.New(ErrTxBeginFailed).
			WithDetail("table", s.TableName).
			WithCause(err)
	}
	defer tx.Rollback()

	// Update each item in the transaction
	for _, item := range items {
		// Extract ID value using reflection
		idValue := extractIDValue(item, s.IDColumn)
		if idValue == nil {
			continue
		}

		// Extract fields and values
		fields, values, err := extractFieldsAndValues(item)
		if err != nil {
			return err
		}

		// Build SET clause
		setClause := buildSetClause(fields)

		// Add ID to values for WHERE clause
		values = append(values, idValue)

		// Build query
		query := fmt.Sprintf(
			"UPDATE %s SET %s WHERE %s = $%d",
			s.TableName,
			setClause,
			s.IDColumn,
			len(values),
		)

		// Execute query in transaction
		_, err = tx.ExecContext(ctx, query, values...)
		if err != nil {
			return storeErrors.New(ErrSQLExecFailed).
				WithDetail("id", fmt.Sprintf("%v", idValue)).
				WithDetail("table", s.TableName).
				WithCause(err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return storeErrors.New(ErrTxCommitFailed).
			WithDetail("table", s.TableName).
			WithCause(err)
	}

	return nil
}

// BulkDelete removes multiple records in a single operation
func (s *TypedSQL[T]) BulkDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Ensure table name is set
	if s.TableName == "" {
		return storeErrors.New(ErrInvalidQuery).
			WithDetail("reason", "table name not set")
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))

	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	// Build query
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s IN (%s)",
		s.TableName,
		s.IDColumn,
		strings.Join(placeholders, ", "),
	)

	// Execute query
	result, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return storeErrors.New(ErrSQLExecFailed).
			WithDetail("operation", "bulk_delete").
			WithDetail("count", len(ids)).
			WithDetail("table", s.TableName).
			WithCause(err)
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return storeErrors.New(ErrSQLExecFailed).
			WithDetail("operation", "bulk_delete").
			WithDetail("table", s.TableName).
			WithCause(err)
	}

	if rowsAffected == 0 {
		return storeErrors.New(ErrRecordNotFound).
			WithDetail("operation", "bulk_delete").
			WithDetail("table", s.TableName)
	}

	return nil
}

// Search performs a full-text search
func (s *TypedSQL[T]) Search(ctx context.Context, query string, opts SearchOptions) ([]T, error) {
	var results []T

	// Ensure table name is set
	if s.TableName == "" {
		return results, storeErrors.New(ErrInvalidQuery).
			WithDetail("reason", "table name not set")
	}

	// Verify search fields are specified
	if len(opts.Fields) == 0 {
		return results, storeErrors.New(ErrInvalidQuery).
			WithDetail("reason", "search fields not specified")
	}

	// Build search conditions for each field
	conditions := make([]string, len(opts.Fields))
	args := []interface{}{query}

	for i, field := range opts.Fields {
		conditions[i] = fmt.Sprintf("%s ILIKE $1", field)
	}

	// Combine conditions with OR
	whereClause := fmt.Sprintf("WHERE %s", strings.Join(conditions, " OR "))

	// Build query with limit and offset
	sqlQuery := fmt.Sprintf("SELECT * FROM %s %s", s.TableName, whereClause)

	if opts.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	if opts.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	// Execute query
	rows, err := s.DB.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return results, storeErrors.New(ErrSQLQueryFailed).
			WithDetail("query", query).
			WithDetail("table", s.TableName).
			WithCause(err)
	}
	defer rows.Close()

	// Scan results
	for rows.Next() {
		var item T
		if err := scanIntoStruct(rows, &item); err != nil {
			return results, storeErrors.New(ErrSQLScanFailed).
				WithDetail("query", query).
				WithDetail("table", s.TableName).
				WithCause(err)
		}
		results = append(results, item)
	}

	// Check for errors after iteration
	if err := rows.Err(); err != nil {
		return results, storeErrors.New(ErrSQLQueryFailed).
			WithDetail("query", query).
			WithDetail("table", s.TableName).
			WithCause(err)
	}

	return results, nil
}

// WithTransaction executes a function within a transaction
func (s *TypedSQL[T]) WithTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	// Begin transaction
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return storeErrors.New(ErrTxBeginFailed).WithCause(err)
	}
	defer tx.Rollback()

	// Execute function within transaction
	if err := fn(ctx); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return storeErrors.New(ErrTxCommitFailed).WithCause(err)
	}

	return nil
}

// PaginateSQL is a generic function for SQL pagination
func PaginateSQL[T any](
	ctx context.Context,
	db *sql.DB,
	opts PaginationOptions,
	baseQuery string,
	countQuery string,
	args []interface{},
	scanFn func(*sql.Rows) (T, error),
) (Paginated[T], error) {

	// If count query is empty, create it from the base query
	if countQuery == "" {
		// Extract the FROM part and everything after
		fromIdx := strings.Index(strings.ToUpper(baseQuery), "FROM")
		if fromIdx == -1 {
			return Paginated[T]{}, storeErrors.NewWithMessage(ErrInvalidQuery, "cannot extract FROM clause")
		}

		// Create a count query by replacing SELECT ... FROM with SELECT COUNT(*) FROM
		countQuery = "SELECT COUNT(*) " + baseQuery[fromIdx:]

		// Remove any ORDER BY clauses from the count query
		orderByIdx := strings.Index(strings.ToUpper(countQuery), "ORDER BY")
		if orderByIdx != -1 {
			countQuery = countQuery[:orderByIdx]
		}
	}

	// 1. Get the total count
	var total int
	err := db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return Paginated[T]{}, storeErrors.NewWithCause(ErrSQLCountFailed, err)
	}

	// 2. Add pagination to the base query
	orderDirection := "ASC"
	if opts.Desc {
		orderDirection = "DESC"
	}

	// Check if the query already has an ORDER BY clause
	hasOrderBy := strings.Contains(strings.ToUpper(baseQuery), "ORDER BY")

	paginatedQuery := baseQuery
	if !hasOrderBy && opts.OrderBy != "" {
		paginatedQuery += fmt.Sprintf(" ORDER BY %s %s", opts.OrderBy, orderDirection)
	}

	// Add LIMIT and OFFSET
	offset := (opts.Page - 1) * opts.PageSize
	paginatedQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", opts.PageSize, offset)

	// 3. Execute the query with pagination
	rows, err := db.QueryContext(ctx, paginatedQuery, args...)
	if err != nil {
		return Paginated[T]{}, storeErrors.NewWithCause(ErrSQLQueryFailed, err)
	}
	defer rows.Close()

	// 4. Process the results using the provided scan function
	var results []T
	for rows.Next() {
		item, err := scanFn(rows)
		if err != nil {
			return Paginated[T]{}, storeErrors.NewWithCause(ErrSQLScanFailed, err)
		}
		results = append(results, item)
	}

	// Check for errors after iteration
	if err = rows.Err(); err != nil {
		return Paginated[T]{}, storeErrors.NewWithCause(ErrSQLQueryFailed, err)
	}

	// 5. Create and return the paginated result
	return NewPaginated(results, opts.Page, opts.PageSize, total), nil
}

// PaginateSimple is a generic function for simple SQL pagination with reflection
func PaginateSimple[T any](
	ctx context.Context,
	db *sql.DB,
	opts PaginationOptions,
	query string,
	args []interface{},
) (Paginated[T], error) {
	// Create a scan function that uses reflection
	scanFn := func(rows *sql.Rows) (T, error) {
		var result T
		err := scanIntoStruct(rows, &result)
		return result, err
	}

	return PaginateSQL(ctx, db, opts, query, "", args, scanFn)
}

// Helper functions

// scanIntoStruct scans a row into a struct using reflection
func scanIntoStruct(scanner interface{}, dest interface{}) error {
	// Implementation would use reflection to scan into struct
	// For simplicity, this is a placeholder
	var row *sql.Row
	var rows *sql.Rows

	switch r := scanner.(type) {
	case *sql.Row:
		row = r
	case *sql.Rows:
		rows = r
	default:
		return fmt.Errorf("invalid scanner type")
	}

	// Get the type information
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer")
	}

	destElem := destValue.Elem()
	if destElem.Kind() != reflect.Struct {
		return fmt.Errorf("destination must be a struct")
	}

	// Use reflection to scan into struct
	// This is a placeholder implementation
	if row != nil {
		return fmt.Errorf("row scanning not implemented")
	} else if rows != nil {
		return fmt.Errorf("rows scanning not implemented")
	}

	return nil
}

// extractFieldsAndValues extracts field names and values from a struct
func extractFieldsAndValues(item interface{}) ([]string, []interface{}, error) {
	// Implementation would use reflection to extract fields and values
	// For simplicity, this is a placeholder
	return nil, nil, nil
}

// buildWhereClause builds a WHERE clause from a map of filters
func buildWhereClause(filters map[string]any) (string, []interface{}) {
	if len(filters) == 0 {
		return "", nil
	}

	conditions := make([]string, 0, len(filters))
	args := make([]interface{}, 0, len(filters))
	i := 1

	for field, value := range filters {
		conditions = append(conditions, fmt.Sprintf("%s = $%d", field, i))
		args = append(args, value)
		i++
	}

	return fmt.Sprintf("WHERE %s", strings.Join(conditions, " AND ")), args
}

// buildSetClause builds a SET clause for UPDATE statements
func buildSetClause(fields []string) string {
	setClauses := make([]string, len(fields))
	for i, field := range fields {
		setClauses[i] = fmt.Sprintf("%s = $%d", field, i+1)
	}
	return strings.Join(setClauses, ", ")
}

// createPlaceholders creates a comma-separated list of SQL placeholders
func createPlaceholders(count int) string {
	placeholders := make([]string, count)
	for i := 0; i < count; i++ {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	return strings.Join(placeholders, ", ")
}

