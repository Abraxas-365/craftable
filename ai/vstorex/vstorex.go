package vstorex

import (
	"context"
	"fmt"
	"time"

	"github.com/Abraxas-365/craftable/ai/document"
	"github.com/Abraxas-365/craftable/ai/embedding"
	"github.com/Abraxas-365/craftable/ai/splitterx"
	"github.com/Abraxas-365/craftable/errx"
	"github.com/google/uuid"
)

// Error registry for vector store-specific errors
var (
	ErrRegistry = errx.NewRegistry("VSTORE")

	// Input validation errors
	ErrCodeInvalidDocument  = ErrRegistry.Register("INVALID_DOCUMENT", errx.TypeValidation, 400, "Invalid document format")
	ErrCodeInvalidVector    = ErrRegistry.Register("INVALID_VECTOR", errx.TypeValidation, 400, "Invalid vector dimensions")
	ErrCodeInvalidFilter    = ErrRegistry.Register("INVALID_FILTER", errx.TypeValidation, 400, "Invalid filter expression")
	ErrCodeInvalidParameter = ErrRegistry.Register("INVALID_PARAMETER", errx.TypeValidation, 400, "Invalid parameter value")

	// Resource errors
	ErrCodeDocumentNotFound   = ErrRegistry.Register("DOCUMENT_NOT_FOUND", errx.TypeNotFound, 404, "Document not found")
	ErrCodeCollectionNotFound = ErrRegistry.Register("COLLECTION_NOT_FOUND", errx.TypeNotFound, 404, "Collection not found")

	// Operational errors
	ErrCodeStoreFailure    = ErrRegistry.Register("STORE_FAILURE", errx.TypeSystem, 500, "Vector store operation failed")
	ErrCodeInitFailure     = ErrRegistry.Register("INIT_FAILURE", errx.TypeSystem, 500, "Failed to initialize vector store")
	ErrCodeConnectionError = ErrRegistry.Register("CONNECTION_ERROR", errx.TypeUnavailable, 503, "Failed to connect to vector database")

	// Processing errors
	ErrCodeEmbeddingFailure = ErrRegistry.Register("EMBEDDING_FAILURE", errx.TypeSystem, 500, "Failed to generate embeddings")
	ErrCodeSplitterFailure  = ErrRegistry.Register("SPLITTER_FAILURE", errx.TypeSystem, 500, "Failed to split document")
	ErrCodeSearchFailure    = ErrRegistry.Register("SEARCH_FAILURE", errx.TypeSystem, 500, "Failed to search vector store")

	// Concurrent operation errors
	ErrCodeConflict = ErrRegistry.Register("CONFLICT", errx.TypeConflict, 409, "Concurrent operation conflict")

	// Rate limiting
	ErrCodeRateLimitExceeded = ErrRegistry.Register("RATE_LIMIT_EXCEEDED", errx.TypeRateLimit, 429, "Rate limit exceeded")
)

// Filter defines the interface for query filters
type Filter interface {
	// ToMap converts the filter to a map representation
	ToMap() map[string]interface{}
}

// MapFilter is a simple map-based filter implementation
type MapFilter map[string]interface{}

func (f MapFilter) ToMap() map[string]interface{} {
	return f
}

// Document extends document.Document with vector store specific fields
type Document struct {
	ID          string                 `json:"id"`
	PageContent string                 `json:"page_content"`
	Metadata    map[string]interface{} `json:"metadata"`
	Vector      []float32              `json:"vector,omitempty"`
	Score       float32                `json:"score,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ToDocument converts a vectorstore.Document to document.Document
func (d Document) ToDocument() *document.Document {
	return &document.Document{
		ID:          d.ID,
		PageContent: d.PageContent,
		Metadata:    d.Metadata,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// FromDocument creates a vectorstore.Document from document.Document
func FromDocument(doc *document.Document) Document {
	return Document{
		ID:          doc.ID,
		PageContent: doc.PageContent,
		Metadata:    doc.Metadata,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
}

// SearchOptions contains options for similarity search
type SearchOptions struct {
	Limit          int     `json:"limit"`
	Offset         int     `json:"offset"`
	Filter         Filter  `json:"filter,omitempty"`
	ScoreThreshold float32 `json:"score_threshold"`
	IncludeVectors bool    `json:"include_vectors"`
	Namespace      string  `json:"namespace,omitempty"`
}

// SearchResult contains search results with pagination info
type SearchResult struct {
	Documents []Document `json:"documents"`
	Total     int        `json:"total"`
	HasMore   bool       `json:"has_more"`
}

// Store defines the interface for vector store implementations
type Store interface {
	// Initialize the store with options
	Initialize(ctx context.Context, options map[string]interface{}) error

	// AddDocuments adds documents with their vectors to the store
	AddDocuments(ctx context.Context, docs []Document, vectors [][]float32) error

	// SimilaritySearch searches for similar documents
	SimilaritySearch(ctx context.Context, vector []float32, options *SearchOptions) (*SearchResult, error)

	// GetDocuments retrieves documents by their IDs
	GetDocuments(ctx context.Context, ids []string) ([]Document, error)

	// DeleteDocuments removes documents by their IDs
	DeleteDocuments(ctx context.Context, ids []string) error

	// DeleteByFilter removes documents matching the filter
	DeleteByFilter(ctx context.Context, filter Filter) error

	// DocumentExists checks if documents exist
	DocumentExists(ctx context.Context, ids []string) ([]bool, error)

	// Count returns the number of documents matching a filter
	Count(ctx context.Context, filter Filter) (int, error)

	// Close the store connections
	Close() error

	// Features returns supported features
	Features() map[string]bool
}

// VectorStore combines a store implementation with an embedder
type VectorStore struct {
	store    Store
	embedder embedding.Embedder
	splitter splitterx.Splitter
	options  *Options
}

// ProcessOptions defines how documents are processed during addition
type ProcessOptions struct {
	// Whether to split documents before adding
	SplitDocuments bool

	// Batch size for processing documents
	BatchSize int

	// Preprocessing function applied to documents before embedding
	Preprocessor func(string) string
}

// AddDocumentOptions contains options for adding documents
type AddDocumentOptions struct {
	ProcessOptions

	// Additional metadata to apply to all documents
	AdditionalMetadata map[string]interface{}

	// Whether to generate IDs for documents that don't have them
	GenerateIDs bool

	// Namespace for multi-tenancy
	Namespace string
}

// CreateVectorStore creates a new vector store with the given components
func CreateVectorStore(
	store Store,
	embedder embedding.Embedder,
	splitter splitterx.Splitter,
	options ...Option,
) *VectorStore {
	opts := DefaultOptions()
	for _, opt := range options {
		opt(opts)
	}

	return &VectorStore{
		store:    store,
		embedder: embedder,
		splitter: splitter,
		options:  opts,
	}
}

// Initialize initializes the vector store
func (vs *VectorStore) Initialize(ctx context.Context) error {
	err := vs.store.Initialize(ctx, vs.options.StoreOptions)
	if err != nil {
		return ErrRegistry.NewWithCause(ErrCodeInitFailure, err).
			WithDetail("store_type", fmt.Sprintf("%T", vs.store))
	}
	return nil
}

// validateDocuments performs validation on documents before processing
func (vs *VectorStore) validateDocuments(docs []document.Document) error {
	if len(docs) == 0 {
		return ErrRegistry.New(ErrCodeInvalidDocument).
			WithDetail("reason", "empty document list")
	}

	for i, doc := range docs {
		if doc.PageContent == "" {
			return ErrRegistry.New(ErrCodeInvalidDocument).
				WithDetails(map[string]interface{}{
					"index":  i,
					"reason": "empty page content",
				})
		}
	}

	return nil
}

// AddDocuments adds documents to the vector store
func (vs *VectorStore) AddDocuments(
	ctx context.Context,
	docs []document.Document,
	opts *AddDocumentOptions,
) error {
	// Validate input documents
	if err := vs.validateDocuments(docs); err != nil {
		return err
	}

	if opts == nil {
		opts = &AddDocumentOptions{
			ProcessOptions: ProcessOptions{
				SplitDocuments: true,
				BatchSize:      100,
			},
			GenerateIDs: true,
		}
	}

	var processedDocs []document.Document

	// Split documents if needed
	if opts.SplitDocuments && vs.splitter != nil {
		var err error
		processedDocs, err = vs.splitter.SplitDocuments(docs)
		if err != nil {
			return ErrRegistry.NewWithCause(ErrCodeSplitterFailure, err).
				WithDetail("splitter_type", fmt.Sprintf("%T", vs.splitter))
		}
	} else {
		processedDocs = docs
	}

	// Convert to vector store documents and generate IDs
	vsDocs := make([]Document, len(processedDocs))
	for i, doc := range processedDocs {
		vsDoc := Document{
			PageContent: doc.PageContent,
			Metadata:    make(map[string]interface{}),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Copy metadata
		for k, v := range doc.Metadata {
			vsDoc.Metadata[k] = v
		}

		// Add additional metadata if provided
		if opts.AdditionalMetadata != nil {
			for k, v := range opts.AdditionalMetadata {
				vsDoc.Metadata[k] = v
			}
		}

		// Add namespace to metadata if provided
		if opts.Namespace != "" {
			vsDoc.Metadata["namespace"] = opts.Namespace
		} else if vs.options.Namespace != "" {
			vsDoc.Metadata["namespace"] = vs.options.Namespace
		}

		// Generate ID if needed
		if opts.GenerateIDs || doc.ID == "" {
			vsDoc.ID = uuid.New().String()
		} else {
			vsDoc.ID = doc.ID
		}

		vsDocs[i] = vsDoc
	}

	// Process in batches
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(vsDocs); i += batchSize {
		end := i + batchSize
		if end > len(vsDocs) {
			end = len(vsDocs)
		}

		batch := vsDocs[i:end]

		// Extract texts for embedding
		texts := make([]string, len(batch))
		for j, doc := range batch {
			// Apply preprocessor if available
			if opts.Preprocessor != nil {
				texts[j] = opts.Preprocessor(doc.PageContent)
			} else {
				texts[j] = doc.PageContent
			}
		}

		// Generate embeddings
		embeddings, err := vs.embedder.EmbedDocuments(ctx, texts)
		if err != nil {
			return ErrRegistry.NewWithCause(ErrCodeEmbeddingFailure, err).
				WithDetails(map[string]interface{}{
					"batch_start": i,
					"batch_size":  len(batch),
				})
		}

		// Extract vectors from embeddings
		vectors := make([][]float32, len(embeddings))
		for j, emb := range embeddings {
			vectors[j] = emb.Vector
		}

		// Store documents with vectors
		err = vs.store.AddDocuments(ctx, batch, vectors)
		if err != nil {
			return ErrRegistry.NewWithCause(ErrCodeStoreFailure, err).
				WithDetails(map[string]interface{}{
					"batch_start": i,
					"batch_size":  len(batch),
					"operation":   "add_documents",
				})
		}
	}

	return nil
}

// SimilaritySearch searches for similar documents
func (vs *VectorStore) SimilaritySearch(
	ctx context.Context,
	query string,
	options *SearchOptions,
) (*SearchResult, error) {
	if query == "" {
		return nil, ErrRegistry.New(ErrCodeInvalidParameter).
			WithDetail("parameter", "query").
			WithDetail("reason", "empty query string")
	}

	if options == nil {
		options = &SearchOptions{
			Limit:          10,
			ScoreThreshold: vs.options.ScoreThreshold,
		}
	}

	// Apply namespace if not specified in options
	if options.Namespace == "" && vs.options.Namespace != "" {
		options.Namespace = vs.options.Namespace
	}

	// Generate embedding for query
	embedding, err := vs.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, ErrRegistry.NewWithCause(ErrCodeEmbeddingFailure, err).
			WithDetail("query", query)
	}

	// Apply default filters if any
	if vs.options.DefaultFilter != nil && options.Filter == nil {
		options.Filter = vs.options.DefaultFilter
	}

	// If there's a namespace, add it to the filter
	if options.Namespace != "" {
		namespaceFilter := MapFilter{"namespace": options.Namespace}
		if options.Filter != nil {
			// Combine both filters with AND logic
			options.Filter = CompoundFilter{
				And: []Filter{namespaceFilter, options.Filter},
			}
		} else {
			options.Filter = namespaceFilter
		}
	}

	// Perform search
	result, err := vs.store.SimilaritySearch(ctx, embedding.Vector, options)
	if err != nil {
		return nil, ErrRegistry.NewWithCause(ErrCodeSearchFailure, err).
			WithDetails(map[string]interface{}{
				"query":      query,
				"limit":      options.Limit,
				"has_filter": options.Filter != nil,
			})
	}

	// Apply score threshold
	if options.ScoreThreshold > 0 {
		filteredDocs := make([]Document, 0, len(result.Documents))
		for _, doc := range result.Documents {
			if doc.Score >= options.ScoreThreshold {
				filteredDocs = append(filteredDocs, doc)
			}
		}
		result.Documents = filteredDocs
		result.HasMore = result.Total > options.Offset+len(filteredDocs)
	}

	return result, nil
}

// GetDocuments retrieves documents by their IDs
func (vs *VectorStore) GetDocuments(ctx context.Context, ids []string) ([]Document, error) {
	if len(ids) == 0 {
		return nil, ErrRegistry.New(ErrCodeInvalidParameter).
			WithDetail("parameter", "ids").
			WithDetail("reason", "empty ID list")
	}

	docs, err := vs.store.GetDocuments(ctx, ids)
	if err != nil {
		return nil, ErrRegistry.NewWithCause(ErrCodeStoreFailure, err).
			WithDetails(map[string]interface{}{
				"ids":       ids,
				"operation": "get_documents",
			})
	}

	if len(docs) == 0 {
		return nil, ErrRegistry.New(ErrCodeDocumentNotFound).
			WithDetail("ids", ids)
	}

	return docs, nil
}

// DeleteDocuments removes documents by their IDs
func (vs *VectorStore) DeleteDocuments(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return ErrRegistry.New(ErrCodeInvalidParameter).
			WithDetail("parameter", "ids").
			WithDetail("reason", "empty ID list")
	}

	err := vs.store.DeleteDocuments(ctx, ids)
	if err != nil {
		return ErrRegistry.NewWithCause(ErrCodeStoreFailure, err).
			WithDetails(map[string]interface{}{
				"ids":       ids,
				"operation": "delete_documents",
			})
	}
	return nil
}

// DeleteByFilter removes documents matching the filter
func (vs *VectorStore) DeleteByFilter(ctx context.Context, filter Filter) error {
	if filter == nil {
		return ErrRegistry.New(ErrCodeInvalidParameter).
			WithDetail("parameter", "filter").
			WithDetail("reason", "nil filter")
	}

	// If there's a namespace, add it to the filter
	if vs.options.Namespace != "" {
		namespaceFilter := MapFilter{"namespace": vs.options.Namespace}
		// Combine both filters with AND logic
		filter = CompoundFilter{
			And: []Filter{namespaceFilter, filter},
		}
	}

	err := vs.store.DeleteByFilter(ctx, filter)
	if err != nil {
		return ErrRegistry.NewWithCause(ErrCodeStoreFailure, err).
			WithDetails(map[string]interface{}{
				"filter":    filter.ToMap(),
				"operation": "delete_by_filter",
			})
	}
	return nil
}

// DocumentExists checks if documents exist
func (vs *VectorStore) DocumentExists(ctx context.Context, ids []string) ([]bool, error) {
	if len(ids) == 0 {
		return nil, ErrRegistry.New(ErrCodeInvalidParameter).
			WithDetail("parameter", "ids").
			WithDetail("reason", "empty ID list")
	}

	exists, err := vs.store.DocumentExists(ctx, ids)
	if err != nil {
		return nil, ErrRegistry.NewWithCause(ErrCodeStoreFailure, err).
			WithDetails(map[string]interface{}{
				"ids":       ids,
				"operation": "document_exists",
			})
	}
	return exists, nil
}

// Count returns the number of documents matching a filter
func (vs *VectorStore) Count(ctx context.Context, filter Filter) (int, error) {
	// If there's a namespace, add it to the filter
	if vs.options.Namespace != "" && filter != nil {
		namespaceFilter := MapFilter{"namespace": vs.options.Namespace}
		// Combine both filters with AND logic
		filter = CompoundFilter{
			And: []Filter{namespaceFilter, filter},
		}
	} else if vs.options.Namespace != "" {
		filter = MapFilter{"namespace": vs.options.Namespace}
	}

	count, err := vs.store.Count(ctx, filter)
	if err != nil {
		details := map[string]interface{}{
			"operation": "count",
		}

		if filter != nil {
			details["filter"] = filter.ToMap()
		} else {
			details["filter"] = nil
		}

		return 0, ErrRegistry.NewWithCause(ErrCodeStoreFailure, err).
			WithDetails(details)
	}
	return count, nil
}

// Close the vector store
func (vs *VectorStore) Close() error {
	err := vs.store.Close()
	if err != nil {
		return ErrRegistry.NewWithCause(ErrCodeConnectionError, err).
			WithDetail("operation", "close")
	}
	return nil
}

// Features returns the store's supported features
func (vs *VectorStore) Features() map[string]bool {
	return vs.store.Features()
}

// Options contains configuration for the vector store
type Options struct {
	// DefaultFilter is applied to all searches if no filter is provided
	DefaultFilter Filter

	// ScoreThreshold is the default minimum similarity score
	ScoreThreshold float32

	// Namespace for multi-tenant support
	Namespace string

	// StoreOptions are passed to the store implementation
	StoreOptions map[string]interface{}
}

// Option is a function that configures Options
type Option func(*Options)

// WithDefaultFilter sets the default filter
func WithDefaultFilter(filter Filter) Option {
	return func(o *Options) {
		o.DefaultFilter = filter
	}
}

// WithScoreThreshold sets the default similarity score threshold
func WithScoreThreshold(threshold float32) Option {
	return func(o *Options) {
		o.ScoreThreshold = threshold
	}
}

// WithNamespace sets the namespace
func WithNamespace(namespace string) Option {
	return func(o *Options) {
		o.Namespace = namespace
	}
}

// WithStoreOption adds a store-specific option
func WithStoreOption(key string, value interface{}) Option {
	return func(o *Options) {
		if o.StoreOptions == nil {
			o.StoreOptions = make(map[string]interface{})
		}
		o.StoreOptions[key] = value
	}
}

// DefaultOptions creates default options
func DefaultOptions() *Options {
	return &Options{
		ScoreThreshold: 0.0,
		StoreOptions:   make(map[string]interface{}),
	}
}

// CompoundFilter for complex filtering with logical operators
type CompoundFilter struct {
	And []Filter `json:"and,omitempty"`
	Or  []Filter `json:"or,omitempty"`
	Not Filter   `json:"not,omitempty"`
}

func (f CompoundFilter) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	if len(f.And) > 0 {
		andFilters := make([]map[string]interface{}, len(f.And))
		for i, filter := range f.And {
			andFilters[i] = filter.ToMap()
		}
		result["$and"] = andFilters
	}

	if len(f.Or) > 0 {
		orFilters := make([]map[string]interface{}, len(f.Or))
		for i, filter := range f.Or {
			orFilters[i] = filter.ToMap()
		}
		result["$or"] = orFilters
	}

	if f.Not != nil {
		result["$not"] = f.Not.ToMap()
	}

	return result
}

// Validate ensures the filter is properly structured
func (f CompoundFilter) Validate() error {
	// Check that at least one condition is specified
	if len(f.And) == 0 && len(f.Or) == 0 && f.Not == nil {
		return ErrRegistry.New(ErrCodeInvalidFilter).
			WithDetail("reason", "compound filter must have at least one condition")
	}

	// Validate And conditions
	for i, filter := range f.And {
		if filter == nil {
			return ErrRegistry.New(ErrCodeInvalidFilter).
				WithDetails(map[string]interface{}{
					"reason": "nil filter in 'and' array",
					"index":  i,
				})
		}

		// Recursively validate if it's a compound filter
		if cf, ok := filter.(interface{ Validate() error }); ok {
			if err := cf.Validate(); err != nil {
				return errx.Wrap(err, "invalid 'and' filter", errx.TypeValidation)
			}
		}
	}

	// Validate Or conditions
	for i, filter := range f.Or {
		if filter == nil {
			return ErrRegistry.New(ErrCodeInvalidFilter).
				WithDetails(map[string]interface{}{
					"reason": "nil filter in 'or' array",
					"index":  i,
				})
		}

		// Recursively validate if it's a compound filter
		if cf, ok := filter.(interface{ Validate() error }); ok {
			if err := cf.Validate(); err != nil {
				return errx.Wrap(err, "invalid 'or' filter", errx.TypeValidation)
			}
		}
	}

	// Validate Not condition
	if f.Not != nil {
		if cf, ok := f.Not.(interface{ Validate() error }); ok {
			if err := cf.Validate(); err != nil {
				return errx.Wrap(err, "invalid 'not' filter", errx.TypeValidation)
			}
		}
	}

	return nil
}

// ComparisonFilter for field comparisons
type ComparisonFilter struct {
	Field string      `json:"field"`
	Op    string      `json:"op"` // eq, neq, gt, gte, lt, lte, in, nin, contains
	Value interface{} `json:"value"`
}

func (f ComparisonFilter) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"field": f.Field,
		"op":    f.Op,
		"value": f.Value,
	}
}

// Validate ensures the comparison filter is properly structured
func (f ComparisonFilter) Validate() error {
	if f.Field == "" {
		return ErrRegistry.New(ErrCodeInvalidFilter).
			WithDetail("reason", "empty field name in comparison filter")
	}

	// Validate operator
	validOps := map[string]bool{
		"eq": true, "neq": true,
		"gt": true, "gte": true,
		"lt": true, "lte": true,
		"in": true, "nin": true,
		"contains": true, "exists": true,
	}

	if !validOps[f.Op] {
		return ErrRegistry.New(ErrCodeInvalidFilter).
			WithDetails(map[string]interface{}{
				"reason":          "invalid operator in comparison filter",
				"operator":        f.Op,
				"field":           f.Field,
				"valid_operators": []string{"eq", "neq", "gt", "gte", "lt", "lte", "in", "nin", "contains", "exists"},
			})
	}

	// Validate value for specific operators
	if f.Op == "in" || f.Op == "nin" {
		// Check if value is a slice
		switch f.Value.(type) {
		case []interface{}, []string, []int, []float64, []bool:
			// Valid slice types
		default:
			return ErrRegistry.New(ErrCodeInvalidFilter).
				WithDetails(map[string]interface{}{
					"reason":     "'in' and 'nin' operators require array values",
					"field":      f.Field,
					"operator":   f.Op,
					"value_type": fmt.Sprintf("%T", f.Value),
				})
		}
	}

	return nil
}

// FilterBuilder helps construct filters with a fluent API
type FilterBuilder struct {
	filter Filter
}

// NewFilterBuilder creates a new filter builder
func NewFilterBuilder() *FilterBuilder {
	return &FilterBuilder{}
}

// And creates a compound filter with AND logic
func (fb *FilterBuilder) And(filters ...Filter) *FilterBuilder {
	// Filter out any nil filters
	nonNilFilters := make([]Filter, 0, len(filters))
	for _, f := range filters {
		if f != nil {
			nonNilFilters = append(nonNilFilters, f)
		}
	}

	if len(nonNilFilters) == 0 {
		return fb
	}

	if len(nonNilFilters) == 1 {
		fb.filter = nonNilFilters[0]
		return fb
	}

	fb.filter = CompoundFilter{And: nonNilFilters}
	return fb
}

// Or creates a compound filter with OR logic
func (fb *FilterBuilder) Or(filters ...Filter) *FilterBuilder {
	// Filter out any nil filters
	nonNilFilters := make([]Filter, 0, len(filters))
	for _, f := range filters {
		if f != nil {
			nonNilFilters = append(nonNilFilters, f)
		}
	}

	if len(nonNilFilters) == 0 {
		return fb
	}

	if len(nonNilFilters) == 1 {
		fb.filter = nonNilFilters[0]
		return fb
	}

	fb.filter = CompoundFilter{Or: nonNilFilters}
	return fb
}

// Not creates a compound filter with NOT logic
func (fb *FilterBuilder) Not(filter Filter) *FilterBuilder {
	if filter == nil {
		return fb
	}

	fb.filter = CompoundFilter{Not: filter}
	return fb
}

// Where creates a comparison filter
func (fb *FilterBuilder) Where(field string, op string, value interface{}) *FilterBuilder {
	fb.filter = ComparisonFilter{
		Field: field,
		Op:    op,
		Value: value,
	}
	return fb
}

// Eq creates an equality comparison filter
func (fb *FilterBuilder) Eq(field string, value interface{}) *FilterBuilder {
	return fb.Where(field, "eq", value)
}

// Neq creates a not-equal comparison filter
func (fb *FilterBuilder) Neq(field string, value interface{}) *FilterBuilder {
	return fb.Where(field, "neq", value)
}

// Gt creates a greater-than comparison filter
func (fb *FilterBuilder) Gt(field string, value interface{}) *FilterBuilder {
	return fb.Where(field, "gt", value)
}

// Gte creates a greater-than-or-equal comparison filter
func (fb *FilterBuilder) Gte(field string, value interface{}) *FilterBuilder {
	return fb.Where(field, "gte", value)
}

// Lt creates a less-than comparison filter
func (fb *FilterBuilder) Lt(field string, value interface{}) *FilterBuilder {
	return fb.Where(field, "lt", value)
}

// Lte creates a less-than-or-equal comparison filter
func (fb *FilterBuilder) Lte(field string, value interface{}) *FilterBuilder {
	return fb.Where(field, "lte", value)
}

// In creates an in-array comparison filter
func (fb *FilterBuilder) In(field string, values interface{}) *FilterBuilder {
	return fb.Where(field, "in", values)
}

// Contains creates a contains-substring comparison filter
func (fb *FilterBuilder) Contains(field string, value string) *FilterBuilder {
	return fb.Where(field, "contains", value)
}

// Exists creates a field-exists filter
func (fb *FilterBuilder) Exists(field string) *FilterBuilder {
	return fb.Where(field, "exists", true)
}

// Build returns the constructed filter
func (fb *FilterBuilder) Build() Filter {
	return fb.filter
}

// Validate ensures the filter is properly structured
func (fb *FilterBuilder) Validate() error {
	if fb.filter == nil {
		return ErrRegistry.New(ErrCodeInvalidFilter).
			WithDetail("reason", "nil filter in filter builder")
	}

	// If the filter has a Validate method, call it
	if validator, ok := fb.filter.(interface{ Validate() error }); ok {
		return validator.Validate()
	}

	return nil
}
