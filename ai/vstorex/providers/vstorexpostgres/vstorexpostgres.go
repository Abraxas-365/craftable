package vectorstorex

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/ai/vstorex"
	"github.com/jmoiron/sqlx"
	"github.com/pgvector/pgvector-go"
)

// PGVectorStoreOptions holds configuration options for the PostgreSQL vector store
type PGVectorStoreOptions struct {
	// ConnectionString is the database connection string
	ConnectionString string

	// TableName is the name of the table to store vectors and documents
	TableName string

	// Dimensions is the dimensionality of the vectors
	Dimensions int

	// IndexType defines the type of index to create (ivfflat, hnsw)
	IndexType string

	// IndexLists defines the number of lists for IVFFlat index (default: 100)
	IndexLists int

	// IndexM defines the max_m parameter for HNSW index (default: 16)
	IndexM int

	// IndexEfConstruction defines the ef_construction parameter for HNSW index (default: 64)
	IndexEfConstruction int

	// DistanceType defines the distance metric (l2, ip, cosine)
	DistanceType string

	// CreateTableIfNotExists automatically creates the table if it doesn't exist
	CreateTableIfNotExists bool

	// CreateIndexIfNotExists automatically creates a vector index if it doesn't exist
	CreateIndexIfNotExists bool

	// EnablePgVectorExtension automatically enables the pgvector extension
	EnablePgVectorExtension bool

	// MaxBatchSize defines the maximum number of documents per batch operation
	MaxBatchSize int

	// MaxConcurrency limits the maximum number of concurrent operations
	MaxConcurrency int
}

// defaultPGVectorStoreOptions creates a default options configuration
func defaultPGVectorStoreOptions() *PGVectorStoreOptions {
	return &PGVectorStoreOptions{
		TableName:               "vector_documents",
		IndexType:               "ivfflat",
		IndexLists:              100,
		IndexM:                  16,
		IndexEfConstruction:     64,
		DistanceType:            "cosine",
		CreateTableIfNotExists:  true,
		CreateIndexIfNotExists:  true,
		EnablePgVectorExtension: true,
		MaxBatchSize:            1000,
		MaxConcurrency:          runtime.NumCPU(),
	}
}

// PGVectorStore provides a PostgreSQL implementation of the Store interface using sqlx
type PGVectorStore struct {
	db      *sqlx.DB
	options *PGVectorStoreOptions
	isOpen  bool
}

// NewPGVectorStore creates a new PostgreSQL vector store with default options
func NewPGVectorStore() *PGVectorStore {
	return &PGVectorStore{
		options: defaultPGVectorStoreOptions(),
		isOpen:  false,
	}
}

// NewPGVectorStoreWithOptions creates a new PostgreSQL vector store with custom options
func NewPGVectorStoreWithOptions(options *PGVectorStoreOptions) *PGVectorStore {
	// Start with defaults
	defaultOpts := defaultPGVectorStoreOptions()

	// Override with provided options
	if options != nil {
		if options.TableName != "" {
			defaultOpts.TableName = options.TableName
		}
		if options.IndexType != "" {
			defaultOpts.IndexType = options.IndexType
		}
		if options.IndexLists > 0 {
			defaultOpts.IndexLists = options.IndexLists
		}
		if options.IndexM > 0 {
			defaultOpts.IndexM = options.IndexM
		}
		if options.IndexEfConstruction > 0 {
			defaultOpts.IndexEfConstruction = options.IndexEfConstruction
		}
		if options.DistanceType != "" {
			defaultOpts.DistanceType = options.DistanceType
		}
		if options.ConnectionString != "" {
			defaultOpts.ConnectionString = options.ConnectionString
		}
		if options.Dimensions > 0 {
			defaultOpts.Dimensions = options.Dimensions
		}

		// Boolean options - explicit setting
		defaultOpts.CreateTableIfNotExists = options.CreateTableIfNotExists
		defaultOpts.CreateIndexIfNotExists = options.CreateIndexIfNotExists
		defaultOpts.EnablePgVectorExtension = options.EnablePgVectorExtension

		if options.MaxBatchSize > 0 {
			defaultOpts.MaxBatchSize = options.MaxBatchSize
		}
		if options.MaxConcurrency > 0 {
			defaultOpts.MaxConcurrency = options.MaxConcurrency
		}
	}

	return &PGVectorStore{
		options: defaultOpts,
		isOpen:  false,
	}
}

// Initialize sets up the PostgreSQL store
func (s *PGVectorStore) Initialize(ctx context.Context, _ map[string]any) error {
	// For backward compatibility, support the old map-based approach
	// But primarily, we expect options to be set through NewPGVectorStoreWithOptions

	if s.options.ConnectionString == "" {
		return fmt.Errorf("%w: missing connection_string in options",
			vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidParameter))
	}

	if s.options.Dimensions <= 0 {
		return fmt.Errorf("%w: dimensions must be greater than 0",
			vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidParameter))
	}

	// Connect to PostgreSQL
	var err error
	s.db, err = sqlx.Connect("postgres", s.options.ConnectionString)
	if err != nil {
		return fmt.Errorf("%w: failed to connect to PostgreSQL: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}

	// Check connection
	if err := s.db.Ping(); err != nil {
		return fmt.Errorf("%w: failed to ping PostgreSQL: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}

	// Enable the pgvector extension if requested
	if s.options.EnablePgVectorExtension {
		_, err = s.db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
		if err != nil {
			return fmt.Errorf("%w: failed to enable vector extension: %v",
				vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
		}
	}

	// Create table if requested
	if s.options.CreateTableIfNotExists {
		createTableSQL := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id TEXT PRIMARY KEY,
				content TEXT,
				metadata JSONB,
				vector vector(%d),
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL
			)
		`, s.options.TableName, s.options.Dimensions)

		_, err = s.db.Exec(createTableSQL)
		if err != nil {
			return fmt.Errorf("%w: failed to create table: %v",
				vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
		}
	}

	// Create index if requested
	if s.options.CreateIndexIfNotExists {
		var indexSQL string
		var opClass string

		// Determine operator class based on distance type
		switch s.options.DistanceType {
		case "l2":
			opClass = "vector_l2_ops"
		case "ip":
			opClass = "vector_ip_ops"
		case "cosine":
			opClass = "vector_cosine_ops"
		default:
			opClass = "vector_cosine_ops"
		}

		// Build index creation SQL based on index type
		switch s.options.IndexType {
		case "hnsw":
			indexSQL = fmt.Sprintf(`
				CREATE INDEX IF NOT EXISTS %s_vector_idx ON %s 
				USING hnsw (vector %s)
				WITH (m = %d, ef_construction = %d)
			`, s.options.TableName, s.options.TableName, opClass,
				s.options.IndexM, s.options.IndexEfConstruction)
		case "ivfflat":
			indexSQL = fmt.Sprintf(`
				CREATE INDEX IF NOT EXISTS %s_vector_idx ON %s 
				USING ivfflat (vector %s)
				WITH (lists = %d)
			`, s.options.TableName, s.options.TableName, opClass, s.options.IndexLists)
		default:
			indexSQL = fmt.Sprintf(`
				CREATE INDEX IF NOT EXISTS %s_vector_idx ON %s 
				USING ivfflat (vector %s)
				WITH (lists = %d)
			`, s.options.TableName, s.options.TableName, opClass, s.options.IndexLists)
		}

		_, err = s.db.Exec(indexSQL)
		if err != nil {
			return fmt.Errorf("%w: failed to create vector index: %v",
				vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
		}
	}

	s.isOpen = true
	return nil
}

// InitializeWithOptions sets up the PostgreSQL store with provided options
func (s *PGVectorStore) InitializeWithOptions(ctx context.Context, options *PGVectorStoreOptions) error {
	// Update store options with the provided options
	if options != nil {
		s.options = options
	}

	// Call the regular initialize method with nil map (we're using the options struct)
	return s.Initialize(ctx, nil)
}

// AddDocuments adds documents with their vectors to the PostgreSQL store
func (s *PGVectorStore) AddDocuments(ctx context.Context, docs []vstorex.Document, vectors [][]float32) error {
	if len(docs) != len(vectors) {
		return fmt.Errorf("%w: documents and vectors count mismatch: %d docs vs %d vectors",
			vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidParameter), len(docs), len(vectors))
	}

	if !s.isOpen {
		return vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure).WithDetail("reason", "store is closed")
	}

	// Check vectors dimensions
	for i, vec := range vectors {
		if len(vec) != s.options.Dimensions {
			return vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidVector).WithDetails(map[string]any{
				"index":        i,
				"expected_dim": s.options.Dimensions,
				"actual_dim":   len(vec),
			})
		}
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to start transaction: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}
	defer tx.Rollback()

	// Prepare the insert statement
	insertSQL := fmt.Sprintf(`
		INSERT INTO %s (id, content, metadata, vector, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			content = EXCLUDED.content,
			metadata = EXCLUDED.metadata,
			vector = EXCLUDED.vector,
			updated_at = EXCLUDED.updated_at
	`, s.options.TableName)

	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		return fmt.Errorf("%w: failed to prepare statement: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}
	defer stmt.Close()

	// Insert documents in batches if needed
	for i := 0; i < len(docs); i += s.options.MaxBatchSize {
		end := i + s.options.MaxBatchSize
		if end > len(docs) {
			end = len(docs)
		}

		batchDocs := docs[i:end]
		batchVectors := vectors[i:end]

		for j, doc := range batchDocs {
			if doc.ID == "" {
				return vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidDocument).WithDetail("index", i+j).
					WithDetail("reason", "document has no ID")
			}

			// Update timestamps
			now := time.Now()
			if doc.CreatedAt.IsZero() {
				doc.CreatedAt = now
			}
			doc.UpdatedAt = now

			// Convert metadata to JSON
			metadataJSON, err := json.Marshal(doc.Metadata)
			if err != nil {
				return fmt.Errorf("%w: failed to marshal metadata: %v",
					vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
			}

			// Create pgvector Vector from float32 slice
			pgvec := pgvector.NewVector(batchVectors[j])

			// Execute insert
			_, err = stmt.ExecContext(ctx, doc.ID, doc.PageContent, metadataJSON, pgvec, doc.CreatedAt, doc.UpdatedAt)
			if err != nil {
				return fmt.Errorf("%w: failed to insert document: %v",
					vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}

	return nil
}

// SimilaritySearch searches for similar documents
func (s *PGVectorStore) SimilaritySearch(ctx context.Context, vector []float32, options *vstorex.SearchOptions) (*vstorex.SearchResult, error) {
	if !s.isOpen {
		return nil, vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure).WithDetail("reason", "store is closed")
	}

	if len(vector) != s.options.Dimensions {
		return nil, vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidVector).WithDetails(map[string]any{
			"expected_dim": s.options.Dimensions,
			"actual_dim":   len(vector),
		})
	}

	// Set defaults
	limit := 10
	offset := 0
	scoreThreshold := float32(0)
	if options != nil {
		if options.Limit > 0 {
			limit = options.Limit
		}
		if options.Offset > 0 {
			offset = options.Offset
		}
		scoreThreshold = options.ScoreThreshold
	}

	// Build query
	var distanceExpr string
	switch s.options.DistanceType {
	case "l2":
		distanceExpr = "1 / (1 + vector <-> $1)" // Convert L2 distance to similarity score
	case "ip":
		distanceExpr = "0.5 + (vector <#> $1) / 2" // Scale inner product to [0,1]
	case "cosine":
		distanceExpr = "1 - (vector <=> $1)" // Cosine similarity is 1 - cosine distance
	default:
		distanceExpr = "1 - (vector <=> $1)" // Default to cosine similarity
	}

	queryParts := []string{
		fmt.Sprintf("SELECT id, content, metadata, vector, created_at, updated_at, %s as score FROM %s",
			distanceExpr, s.options.TableName),
	}

	args := []any{pgvector.NewVector(vector)}
	paramCounter := 2

	// Apply filter if provided
	if options != nil && options.Filter != nil {
		whereSQL, whereArgs, err := buildFilterSQL(options.Filter, paramCounter)
		if err != nil {
			return nil, err
		}

		if whereSQL != "" {
			queryParts = append(queryParts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
			paramCounter += len(whereArgs)
		}

	}

	// Apply score threshold
	if scoreThreshold > 0 {
		whereClause := fmt.Sprintf("%s >= %f", distanceExpr, scoreThreshold)
		if len(queryParts) == 1 {
			// No existing WHERE clause, add one
			queryParts = append(queryParts, "WHERE "+whereClause)
		} else {
			// WHERE clause already exists, add with AND
			queryParts[1] = queryParts[1] + " AND " + whereClause
		}
	}

	// Add ordering and pagination
	queryParts = append(queryParts, "ORDER BY score DESC")
	queryParts = append(queryParts, fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset))

	// Combine everything
	query := strings.Join(queryParts, " ")

	// Count total matching records (without pagination)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", s.options.TableName)
	if len(queryParts) > 1 && strings.HasPrefix(queryParts[1], "WHERE") {
		countQuery += " " + queryParts[1]
	}

	// Execute count query
	var total int
	err := s.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to count documents: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}

	// Execute search query
	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to execute search query: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}
	defer rows.Close()

	// Process results
	var documents []vstorex.Document
	for rows.Next() {
		var doc vstorex.Document
		var metadataJSON []byte
		var pgvec pgvector.Vector
		var score float32

		err := rows.Scan(&doc.ID, &doc.PageContent, &metadataJSON, &pgvec, &doc.CreatedAt, &doc.UpdatedAt, &score)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to scan document: %v",
				vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
		}

		// Parse metadata
		doc.Metadata = make(map[string]any)
		if err := json.Unmarshal(metadataJSON, &doc.Metadata); err != nil {
			return nil, fmt.Errorf("%w: failed to unmarshal metadata: %v",
				vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
		}

		// Add score
		doc.Score = score

		// Include vector if requested
		if options != nil && options.IncludeVectors {
			doc.Vector = pgvec.Slice()
		}

		documents = append(documents, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: error iterating results: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}

	return &vstorex.SearchResult{
		Documents: documents,
		Total:     total,
		HasMore:   offset+len(documents) < total,
	}, nil
}

// buildFilterSQL converts a Filter to SQL WHERE clause and parameters
func buildFilterSQL(filter vstorex.Filter, startParamIndex int) (string, []any, error) {
	if filter == nil {
		return "", nil, nil
	}

	filterMap := filter.ToMap()

	// Handle compound filters
	if andFilters, ok := filterMap["$and"].([]map[string]any); ok {
		clauses := make([]string, 0, len(andFilters))
		params := make([]any, 0)
		paramIndex := startParamIndex

		for _, andFilter := range andFilters {
			subFilter := vstorex.MapFilter(andFilter)
			subClause, subParams, err := buildFilterSQL(subFilter, paramIndex)
			if err != nil {
				return "", nil, err
			}

			if subClause != "" {
				clauses = append(clauses, "("+subClause+")")
				params = append(params, subParams...)
				paramIndex += len(subParams)
			}
		}

		if len(clauses) > 0 {
			return strings.Join(clauses, " AND "), params, nil
		}
		return "", nil, nil
	}

	if orFilters, ok := filterMap["$or"].([]map[string]any); ok {
		clauses := make([]string, 0, len(orFilters))
		params := make([]any, 0)
		paramIndex := startParamIndex

		for _, orFilter := range orFilters {
			subFilter := vstorex.MapFilter(orFilter)
			subClause, subParams, err := buildFilterSQL(subFilter, paramIndex)
			if err != nil {
				return "", nil, err
			}

			if subClause != "" {
				clauses = append(clauses, "("+subClause+")")
				params = append(params, subParams...)
				paramIndex += len(subParams)
			}
		}

		if len(clauses) > 0 {
			return strings.Join(clauses, " OR "), params, nil
		}
		return "", nil, nil
	}

	if notFilter, ok := filterMap["$not"].(map[string]any); ok {
		subFilter := vstorex.MapFilter(notFilter)
		subClause, subParams, err := buildFilterSQL(subFilter, startParamIndex)
		if err != nil {
			return "", nil, err
		}

		if subClause != "" {
			return "NOT (" + subClause + ")", subParams, nil
		}
		return "", nil, nil
	}

	// Handle comparison filters
	if field, ok := filterMap["field"].(string); ok {
		op, hasOp := filterMap["op"].(string)
		value, hasValue := filterMap["value"]

		if hasOp && hasValue {
			return buildComparisonSQL(field, op, value, startParamIndex)
		}
	}

	// Handle simple field-value filters (implicit equality)
	clauses := make([]string, 0)
	params := make([]any, 0)
	paramIndex := startParamIndex

	for field, value := range filterMap {
		// Skip special fields
		if field == "$and" || field == "$or" || field == "$not" {
			continue
		}

		// Handle different value types appropriately
		switch v := value.(type) {
		case int, int32, int64, float32, float64:
			// For numeric values, cast the JSON text to numeric
			clauses = append(clauses, fmt.Sprintf("(metadata->>'%s')::numeric = $%d", field, paramIndex))
			params = append(params, v)
		case bool:
			// For boolean values
			clauses = append(clauses, fmt.Sprintf("metadata->>'%s' = $%d", field, paramIndex))
			params = append(params, fmt.Sprintf("%t", v))
		case string:
			// For string values, direct comparison
			clauses = append(clauses, fmt.Sprintf("metadata->>'%s' = $%d", field, paramIndex))
			params = append(params, v)
		default:
			// For any other type, convert to string
			clauses = append(clauses, fmt.Sprintf("metadata->>'%s' = $%d", field, paramIndex))
			params = append(params, fmt.Sprintf("%v", v))
		}
		paramIndex++
	}

	if len(clauses) > 0 {
		return strings.Join(clauses, " AND "), params, nil
	}

	return "", nil, nil
}

// buildComparisonSQL builds SQL for a comparison operation
func buildComparisonSQL(field, op string, value any, paramIndex int) (string, []any, error) {
	var clause string
	var params []any

	// Handle numeric operations specially
	isNumeric := false
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		isNumeric = true
	}

	switch op {
	case "eq":
		if isNumeric {
			clause = fmt.Sprintf("(metadata->>'%s')::numeric = $%d", field, paramIndex)
			params = []any{value}
		} else {
			clause = fmt.Sprintf("metadata->>'%s' = $%d", field, paramIndex)
			params = []any{fmt.Sprintf("%v", value)}
		}
	case "neq":
		if isNumeric {
			clause = fmt.Sprintf("(metadata->>'%s')::numeric != $%d", field, paramIndex)
			params = []any{value}
		} else {
			clause = fmt.Sprintf("metadata->>'%s' != $%d", field, paramIndex)
			params = []any{fmt.Sprintf("%v", value)}
		}
	case "gt":
		clause = fmt.Sprintf("(metadata->>'%s')::numeric > $%d", field, paramIndex)
		params = []any{value}
	case "gte":
		clause = fmt.Sprintf("(metadata->>'%s')::numeric >= $%d", field, paramIndex)
		params = []any{value}
	case "lt":
		clause = fmt.Sprintf("(metadata->>'%s')::numeric < $%d", field, paramIndex)
		params = []any{value}
	case "lte":
		clause = fmt.Sprintf("(metadata->>'%s')::numeric <= $%d", field, paramIndex)
		params = []any{value}
	case "in":
		if values, ok := value.([]any); ok {
			placeholders := make([]string, len(values))
			params = make([]any, len(values))

			for i, v := range values {
				placeholders[i] = fmt.Sprintf("$%d", paramIndex+i)

				// Handle numeric values in lists
				if _, isNum := v.(float64); isNum || v == nil {
					if isNumeric {
						params[i] = v // Keep as numeric
					} else {
						params[i] = fmt.Sprintf("%v", v) // Convert to string
					}
				} else {
					params[i] = fmt.Sprintf("%v", v)
				}
			}

			if isNumeric {
				clause = fmt.Sprintf("(metadata->>'%s')::numeric IN (%s)", field, strings.Join(placeholders, ", "))
			} else {
				clause = fmt.Sprintf("metadata->>'%s' IN (%s)", field, strings.Join(placeholders, ", "))
			}
		} else {
			return "", nil, vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidFilter).WithDetail("reason", "invalid value for IN operator")
		}
	case "nin":
		if values, ok := value.([]any); ok {
			placeholders := make([]string, len(values))
			params = make([]any, len(values))

			for i, v := range values {
				placeholders[i] = fmt.Sprintf("$%d", paramIndex+i)

				// Handle numeric values in lists
				if _, isNum := v.(float64); isNum || v == nil {
					if isNumeric {
						params[i] = v // Keep as numeric
					} else {
						params[i] = fmt.Sprintf("%v", v) // Convert to string
					}
				} else {
					params[i] = fmt.Sprintf("%v", v)
				}
			}

			if isNumeric {
				clause = fmt.Sprintf("(metadata->>'%s')::numeric NOT IN (%s)", field, strings.Join(placeholders, ", "))
			} else {
				clause = fmt.Sprintf("metadata->>'%s' NOT IN (%s)", field, strings.Join(placeholders, ", "))
			}
		} else {
			return "", nil, vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidFilter).WithDetail("reason", "invalid value for NOT IN operator")
		}
	case "contains":
		clause = fmt.Sprintf("metadata->>'%s' LIKE '%%' || $%d || '%%'", field, paramIndex)
		params = []any{value}
	case "exists":
		if val, ok := value.(bool); ok {
			if val {
				clause = fmt.Sprintf("metadata ? '%s'", field)
			} else {
				clause = fmt.Sprintf("NOT (metadata ? '%s')", field)
			}
			params = []any{}
		} else {
			return "", nil, vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidFilter).WithDetail("reason", "invalid value for EXISTS operator")
		}
	default:
		return "", nil, vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidFilter).WithDetail("reason", "unknown operator: "+op)
	}

	return clause, params, nil
}

// GetDocuments retrieves documents by their IDs
func (s *PGVectorStore) GetDocuments(ctx context.Context, ids []string) ([]vstorex.Document, error) {
	if !s.isOpen {
		return nil, vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure).WithDetail("reason", "store is closed")
	}

	if len(ids) == 0 {
		return nil, vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidParameter).WithDetail("reason", "empty id list")
	}

	// Create placeholders for the query
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	// Build the query
	query := fmt.Sprintf(`
		SELECT id, content, metadata, vector, created_at, updated_at
		FROM %s
		WHERE id IN (%s)
	`, s.options.TableName, strings.Join(placeholders, ", "))

	// Execute the query
	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to retrieve documents: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}
	defer rows.Close()

	// Process results
	var documents []vstorex.Document
	for rows.Next() {
		var doc vstorex.Document
		var metadataJSON []byte
		var pgvec pgvector.Vector

		err := rows.Scan(&doc.ID, &doc.PageContent, &metadataJSON, &pgvec, &doc.CreatedAt, &doc.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to scan document: %v",
				vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
		}

		// Parse metadata
		doc.Metadata = make(map[string]any)
		if err := json.Unmarshal(metadataJSON, &doc.Metadata); err != nil {
			return nil, fmt.Errorf("%w: failed to unmarshal metadata: %v",
				vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
		}

		// Add vector
		doc.Vector = pgvec.Slice()

		documents = append(documents, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: error iterating results: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}

	return documents, nil
}

// DeleteDocuments removes documents by their IDs
func (s *PGVectorStore) DeleteDocuments(ctx context.Context, ids []string) error {
	if !s.isOpen {
		return vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure).WithDetail("reason", "store is closed")
	}

	if len(ids) == 0 {
		return vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidParameter).WithDetail("reason", "empty id list")
	}

	// Create placeholders for the query
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	// Build the query
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE id IN (%s)
	`, s.options.TableName, strings.Join(placeholders, ", "))

	// Execute the query
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%w: failed to delete documents: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}

	return nil
}

// DeleteByFilter removes documents matching the filter
func (s *PGVectorStore) DeleteByFilter(ctx context.Context, filter vstorex.Filter) error {
	if !s.isOpen {
		return vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure).WithDetail("reason", "store is closed")
	}

	if filter == nil {
		return vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidFilter).WithDetail("reason", "nil filter")
	}

	// Build filter SQL
	whereSQL, params, err := buildFilterSQL(filter, 1)
	if err != nil {
		return err
	}

	if whereSQL == "" {
		return vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidFilter).WithDetail("reason", "empty filter")
	}

	// Build the query
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE %s
	`, s.options.TableName, whereSQL)

	// Execute the query
	_, err = s.db.ExecContext(ctx, query, params...)
	if err != nil {
		return fmt.Errorf("%w: failed to delete documents by filter: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}

	return nil
}

// DocumentExists checks if documents exist
func (s *PGVectorStore) DocumentExists(ctx context.Context, ids []string) ([]bool, error) {
	if !s.isOpen {
		return nil, vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure).WithDetail("reason", "store is closed")
	}

	if len(ids) == 0 {
		return nil, vstorex.ErrRegistry.New(vstorex.ErrCodeInvalidParameter).WithDetail("reason", "empty id list")
	}

	// Create a map to track existence
	exists := make([]bool, len(ids))
	idMap := make(map[string]int)
	for i, id := range ids {
		idMap[id] = i
	}

	// Create placeholders for the query
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	// Build the query
	query := fmt.Sprintf(`
		SELECT id
		FROM %s
		WHERE id IN (%s)
	`, s.options.TableName, strings.Join(placeholders, ", "))

	// Execute the query
	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to check document existence: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}
	defer rows.Close()

	// Process results
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("%w: failed to scan document id: %v",
				vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
		}

		if idx, ok := idMap[id]; ok {
			exists[idx] = true
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: error iterating results: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}

	return exists, nil
}

// Count returns the number of documents matching a filter
func (s *PGVectorStore) Count(ctx context.Context, filter vstorex.Filter) (int, error) {
	if !s.isOpen {
		return 0, vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure).WithDetail("reason", "store is closed")
	}

	// Build base query
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", s.options.TableName)

	// Apply filter if provided
	var whereSQL string
	var params []any
	var err error

	if filter != nil {
		whereSQL, params, err = buildFilterSQL(filter, 1)
		if err != nil {
			return 0, err
		}
		if whereSQL != "" {
			query += " WHERE " + whereSQL
		}
	}

	// Execute the query
	var count int
	err = s.db.GetContext(ctx, &count, query, params...)
	if err != nil {
		return 0, fmt.Errorf("%w: failed to count documents: %v",
			vstorex.ErrRegistry.New(vstorex.ErrCodeStoreFailure), err)
	}

	return count, nil
}

// Close the store
func (s *PGVectorStore) Close() error {
	if !s.isOpen {
		return nil
	}

	s.isOpen = false
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Features returns the supported features
func (s *PGVectorStore) Features() map[string]bool {
	return map[string]bool{
		"filter":      true,
		"delete":      true,
		"count":       true,
		"namespace":   true,
		"metadata":    true,
		"exact_match": true,
	}
}
