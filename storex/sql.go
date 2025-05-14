package storex

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
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

// FIXED scanIntoStruct function
func scanIntoStruct(scanner interface{}, dest interface{}) error {
	// Get the value and type of the destination
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer")
	}

	destElem := destValue.Elem()
	destType := destElem.Type()

	// Get all field information from the struct
	numFields := destType.NumField()
	fieldNameMap := make(map[string]int)

	for i := 0; i < numFields; i++ {
		field := destType.Field(i)

		// Check for db tag first
		dbFieldName := field.Tag.Get("db")
		if dbFieldName == "-" {
			continue // Skip this field
		}

		if dbFieldName == "" {
			// If no db tag, use field name (lowercase for matching)
			dbFieldName = strings.ToLower(field.Name)
		}

		fieldNameMap[dbFieldName] = i
	}

	// Process based on scanner type
	switch s := scanner.(type) {
	case *sql.Row:
		// Create a slice of interface{} to scan into
		scanTargets := make([]interface{}, 0, numFields)
		fieldPtrs := make([]interface{}, 0, numFields)

		// Create pointers for all fields
		for i := 0; i < numFields; i++ {
			if !destElem.Field(i).CanSet() {
				continue // Skip unexported fields
			}

			// Create a new value of the field's type
			fieldPtr := reflect.New(destType.Field(i).Type).Interface()
			fieldPtrs = append(fieldPtrs, fieldPtr)
			scanTargets = append(scanTargets, fieldPtr)
		}

		// Scan the row into our targets
		if err := s.Scan(scanTargets...); err != nil {
			return err
		}

		// Copy the scanned values to the destination struct
		fieldIndex := 0
		for i := 0; i < numFields; i++ {
			field := destElem.Field(i)
			if !field.CanSet() {
				continue
			}

			// Get the value that was scanned
			scannedValue := reflect.ValueOf(fieldPtrs[fieldIndex]).Elem()

			// Set the field in our destination struct
			field.Set(scannedValue)

			fieldIndex++
		}

		return nil

	case *sql.Rows:
		// Get the column names
		cols, err := s.Columns()
		if err != nil {
			return err
		}

		// Create a slice of interface{} to scan into
		values := make([]interface{}, len(cols))
		pointers := make([]interface{}, len(cols))

		// For each column, create a pointer to a destination
		for i := range values {
			pointers[i] = &values[i]
		}

		// Scan the row into our targets
		if err := s.Scan(pointers...); err != nil {
			return err
		}

		// For each column, assign to the appropriate struct field
		for i, colName := range cols {
			fieldIdx, ok := fieldNameMap[colName]
			if !ok {
				// Column doesn't match any field, skip it
				continue
			}

			field := destElem.Field(fieldIdx)
			if !field.CanSet() {
				// Field can't be set (unexported), skip it
				continue
			}

			// Get the value from the pointer
			val := values[i]

			// Skip nil values
			if val == nil {
				continue
			}

			// Set the field based on the value's type
			if err := assignValueToField(field, val); err != nil {
				return fmt.Errorf("field %s: %w", destType.Field(fieldIdx).Name, err)
			}
		}

		return nil

	default:
		return fmt.Errorf("unsupported scanner type: %T", scanner)
	}
}

// Helper function to assign a value to a field
func assignValueToField(field reflect.Value, value interface{}) error {
	// Handle nil values
	if value == nil {
		return nil
	}

	fieldType := field.Type()
	valueType := reflect.TypeOf(value)

	// If types are directly assignable
	if valueType.AssignableTo(fieldType) {
		field.Set(reflect.ValueOf(value))
		return nil
	}

	// Handle special cases based on field kind
	switch field.Kind() {
	case reflect.String:
		// Convert value to string
		field.SetString(fmt.Sprintf("%v", value))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Try to convert value to int64
		var intVal int64

		switch v := value.(type) {
		case int:
			intVal = int64(v)
		case int64:
			intVal = v
		case int32:
			intVal = int64(v)
		case int16:
			intVal = int64(v)
		case int8:
			intVal = int64(v)
		case uint:
			intVal = int64(v)
		case uint64:
			intVal = int64(v)
		case uint32:
			intVal = int64(v)
		case uint16:
			intVal = int64(v)
		case uint8:
			intVal = int64(v)
		case float64:
			intVal = int64(v)
		case float32:
			intVal = int64(v)
		case string:
			var err error
			intVal, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("cannot convert string '%s' to int: %w", v, err)
			}
		default:
			return fmt.Errorf("cannot convert %T to int64", value)
		}

		field.SetInt(intVal)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Try to convert value to uint64
		var uintVal uint64

		switch v := value.(type) {
		case uint:
			uintVal = uint64(v)
		case uint64:
			uintVal = v
		case uint32:
			uintVal = uint64(v)
		case uint16:
			uintVal = uint64(v)
		case uint8:
			uintVal = uint64(v)
		case int:
			if v < 0 {
				return fmt.Errorf("cannot convert negative int to uint")
			}
			uintVal = uint64(v)
		case int64:
			if v < 0 {
				return fmt.Errorf("cannot convert negative int64 to uint")
			}
			uintVal = uint64(v)
		case int32:
			if v < 0 {
				return fmt.Errorf("cannot convert negative int32 to uint")
			}
			uintVal = uint64(v)
		case int16:
			if v < 0 {
				return fmt.Errorf("cannot convert negative int16 to uint")
			}
			uintVal = uint64(v)
		case int8:
			if v < 0 {
				return fmt.Errorf("cannot convert negative int8 to uint")
			}
			uintVal = uint64(v)
		case string:
			var err error
			uintVal, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				return fmt.Errorf("cannot convert string '%s' to uint: %w", v, err)
			}
		default:
			return fmt.Errorf("cannot convert %T to uint64", value)
		}

		field.SetUint(uintVal)

	case reflect.Float32, reflect.Float64:
		// Try to convert value to float64
		var floatVal float64

		switch v := value.(type) {
		case float64:
			floatVal = v
		case float32:
			floatVal = float64(v)
		case int:
			floatVal = float64(v)
		case int64:
			floatVal = float64(v)
		case int32:
			floatVal = float64(v)
		case int16:
			floatVal = float64(v)
		case int8:
			floatVal = float64(v)
		case uint:
			floatVal = float64(v)
		case uint64:
			floatVal = float64(v)
		case uint32:
			floatVal = float64(v)
		case uint16:
			floatVal = float64(v)
		case uint8:
			floatVal = float64(v)
		case string:
			var err error
			floatVal, err = strconv.ParseFloat(v, 64)
			if err != nil {
				return fmt.Errorf("cannot convert string '%s' to float: %w", v, err)
			}
		default:
			return fmt.Errorf("cannot convert %T to float64", value)
		}

		field.SetFloat(floatVal)

	case reflect.Bool:
		// Try to convert value to bool
		var boolVal bool

		switch v := value.(type) {
		case bool:
			boolVal = v
		case int:
			boolVal = v != 0
		case int64:
			boolVal = v != 0
		case int32:
			boolVal = v != 0
		case string:
			var err error
			boolVal, err = strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("cannot convert string '%s' to bool: %w", v, err)
			}
		default:
			return fmt.Errorf("cannot convert %T to bool", value)
		}

		field.SetBool(boolVal)

	case reflect.Struct:
		// Special handling for time.Time
		if field.Type() == reflect.TypeOf(time.Time{}) {
			switch v := value.(type) {
			case time.Time:
				field.Set(reflect.ValueOf(v))
			case string:
				// Try to parse time string in different formats
				layouts := []string{
					time.RFC3339,
					"2006-01-02 15:04:05",
					"2006-01-02",
					"15:04:05",
				}

				var parsedTime time.Time
				var err error

				for _, layout := range layouts {
					parsedTime, err = time.Parse(layout, v)
					if err == nil {
						field.Set(reflect.ValueOf(parsedTime))
						return nil
					}
				}

				return fmt.Errorf("cannot parse time from string '%s'", v)
			default:
				return fmt.Errorf("cannot convert %T to time.Time", value)
			}
		} else {
			return fmt.Errorf("cannot handle struct type %s", field.Type().Name())
		}

	case reflect.Ptr:
		// Create a new value to hold the converted value
		elemType := field.Type().Elem()
		newValue := reflect.New(elemType)

		// Set the pointer field to the new value
		field.Set(newValue)

		// Handle the element value
		return assignValueToField(newValue.Elem(), value)

	case reflect.Slice:
		// Handle []byte specially
		if field.Type() == reflect.TypeOf([]byte{}) {
			switch v := value.(type) {
			case []byte:
				field.SetBytes(v)
			case string:
				field.SetBytes([]byte(v))
			default:
				return fmt.Errorf("cannot convert %T to []byte", value)
			}
		} else {
			return fmt.Errorf("cannot handle slice type %s", field.Type().String())
		}

	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
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
