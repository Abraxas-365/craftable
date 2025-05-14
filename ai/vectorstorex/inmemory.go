package vectorstorex

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// InMemoryStore provides an in-memory implementation of the Store interface
type InMemoryStore struct {
	docs       map[string]Document
	vectors    map[string][]float32
	mu         sync.RWMutex
	dimensions int
	isOpen     bool
}

// NewInMemoryStore creates a new in-memory vector store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		docs:    make(map[string]Document),
		vectors: make(map[string][]float32),
		isOpen:  true,
	}
}

// Initialize sets up the in-memory store with options
func (s *InMemoryStore) Initialize(_ context.Context, options map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if dimensions is provided in options
	if dimVal, ok := options["dimensions"]; ok {
		if dim, ok := dimVal.(int); ok {
			s.dimensions = dim
		} else {
			return fmt.Errorf("%w: invalid dimensions parameter: %v",
				errRegistry.New(ErrCodeInvalidParameter), dimVal)
		}
	}

	s.isOpen = true
	return nil
}

// AddDocuments adds documents with their vectors to the in-memory store
func (s *InMemoryStore) AddDocuments(_ context.Context, docs []Document, vectors [][]float32) error {
	if len(docs) != len(vectors) {
		return fmt.Errorf("%w: documents and vectors count mismatch: %d docs vs %d vectors",
			errRegistry.New(ErrCodeInvalidParameter), len(docs), len(vectors))
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isOpen {
		return errRegistry.New(ErrCodeStoreFailure).WithDetail("reason", "store is closed")
	}

	// Check vectors dimensions
	for i, vec := range vectors {
		// If this is the first vector, set dimensions
		if s.dimensions == 0 && len(s.vectors) == 0 {
			s.dimensions = len(vec)
		} else if len(vec) != s.dimensions {
			return errRegistry.New(ErrCodeInvalidVector).WithDetails(map[string]interface{}{
				"index":        i,
				"expected_dim": s.dimensions,
				"actual_dim":   len(vec),
			})
		}
	}

	// Store documents and vectors
	for i, doc := range docs {
		if doc.ID == "" {
			return errRegistry.New(ErrCodeInvalidDocument).WithDetail("index", i).
				WithDetail("reason", "document has no ID")
		}

		// Update timestamps
		now := time.Now()
		if doc.CreatedAt.IsZero() {
			doc.CreatedAt = now
		}
		doc.UpdatedAt = now

		// Store document and its vector
		s.docs[doc.ID] = doc
		s.vectors[doc.ID] = vectors[i]
	}

	return nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// documentMatchesFilter checks if a document matches the specified filter
func documentMatchesFilter(doc Document, filter Filter) bool {
	if filter == nil {
		return true
	}

	filterMap := filter.ToMap()

	// Handle compound filters
	if andFilters, ok := filterMap["$and"].([]map[string]interface{}); ok {
		for _, andFilter := range andFilters {
			// Create a MapFilter for each sub-filter
			subFilter := MapFilter(andFilter)
			if !documentMatchesFilter(doc, subFilter) {
				return false
			}
		}
		return true
	}

	if orFilters, ok := filterMap["$or"].([]map[string]interface{}); ok {
		for _, orFilter := range orFilters {
			// Create a MapFilter for each sub-filter
			subFilter := MapFilter(orFilter)
			if documentMatchesFilter(doc, subFilter) {
				return true
			}
		}
		return false
	}

	if notFilter, ok := filterMap["$not"].(map[string]interface{}); ok {
		subFilter := MapFilter(notFilter)
		return !documentMatchesFilter(doc, subFilter)
	}

	// Handle comparison filters
	if field, ok := filterMap["field"].(string); ok {
		op, hasOp := filterMap["op"].(string)
		value, hasValue := filterMap["value"]

		if hasOp && hasValue {
			return compareFieldValue(doc, field, op, value)
		}
	}

	// Handle simple field-value filters (implicit equality)
	for field, expectedValue := range filterMap {
		// Skip special fields
		if field == "$and" || field == "$or" || field == "$not" {
			continue
		}

		// Check if the field is in metadata
		if actualValue, ok := doc.Metadata[field]; ok {
			if actualValue != expectedValue {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

// compareFieldValue performs a comparison based on the operator
func compareFieldValue(doc Document, field, op string, expected interface{}) bool {
	// Get the actual value from metadata
	actual, ok := doc.Metadata[field]
	if !ok {
		// For exists operator, return false if field doesn't exist
		if op == "exists" {
			return expected.(bool) == false
		}
		return false
	}

	// For exists operator, return true if field exists
	if op == "exists" {
		return expected.(bool) == true
	}

	switch op {
	case "eq":
		return actual == expected
	case "neq":
		return actual != expected
	case "gt", "gte", "lt", "lte":
		return compareNumeric(actual, expected, op)
	case "in":
		return valueInSlice(actual, expected)
	case "nin":
		return !valueInSlice(actual, expected)
	case "contains":
		// If actual is a string and expected is a string, check if actual contains expected
		actualStr, actualIsStr := actual.(string)
		expectedStr, expectedIsStr := expected.(string)

		if actualIsStr && expectedIsStr {
			return strings.Contains(actualStr, expectedStr)
		}
		return false
	default:
		return false
	}
}

// compareNumeric handles numeric comparisons
func compareNumeric(a, b interface{}, op string) bool {
	// Try to convert both to float64 for comparison
	aFloat, aErr := toFloat64(a)
	bFloat, bErr := toFloat64(b)

	if aErr != nil || bErr != nil {
		return false
	}

	switch op {
	case "gt":
		return aFloat > bFloat
	case "gte":
		return aFloat >= bFloat
	case "lt":
		return aFloat < bFloat
	case "lte":
		return aFloat <= bFloat
	default:
		return false
	}
}

// toFloat64 attempts to convert various number types to float64
func toFloat64(v interface{}) (float64, error) {
	switch n := v.(type) {
	case int:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case float32:
		return float64(n), nil
	case float64:
		return n, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

// valueInSlice checks if a value is in a slice
func valueInSlice(value, slice interface{}) bool {
	switch s := slice.(type) {
	case []interface{}:
		for _, item := range s {
			if item == value {
				return true
			}
		}
	case []string:
		if strValue, ok := value.(string); ok {
			for _, item := range s {
				if item == strValue {
					return true
				}
			}
		}
	case []int:
		if intValue, ok := value.(int); ok {
			for _, item := range s {
				if item == intValue {
					return true
				}
			}
		}
	case []float64:
		if floatValue, ok := value.(float64); ok {
			for _, item := range s {
				if item == floatValue {
					return true
				}
			}
		}
	}
	return false
}

// SimilaritySearch searches for similar documents
func (s *InMemoryStore) SimilaritySearch(_ context.Context, vector []float32, options *SearchOptions) (*SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isOpen {
		return nil, errRegistry.New(ErrCodeStoreFailure).WithDetail("reason", "store is closed")
	}

	if len(vector) != s.dimensions && s.dimensions > 0 {
		return nil, errRegistry.New(ErrCodeInvalidVector).WithDetails(map[string]interface{}{
			"expected_dim": s.dimensions,
			"actual_dim":   len(vector),
		})
	}

	// Set defaults
	limit := 10
	offset := 0
	if options != nil {
		if options.Limit > 0 {
			limit = options.Limit
		}
		if options.Offset > 0 {
			offset = options.Offset
		}
	}

	// Calculate similarities and filter documents
	type docWithScore struct {
		doc   Document
		score float32
	}

	var results []docWithScore

	for id, docVector := range s.vectors {
		doc, exists := s.docs[id]
		if !exists {
			continue
		}

		// Apply filter if provided
		if options != nil && options.Filter != nil && !documentMatchesFilter(doc, options.Filter) {
			continue
		}

		// Calculate similarity score
		score := cosineSimilarity(vector, docVector)

		// Apply score threshold if provided
		if options != nil && score < options.ScoreThreshold {
			continue
		}

		// Add document with score to results
		docCopy := doc
		docCopy.Score = score

		// Include vectors if requested
		if options != nil && options.IncludeVectors {
			docCopy.Vector = make([]float32, len(docVector))
			copy(docCopy.Vector, docVector)
		}

		results = append(results, docWithScore{doc: docCopy, score: score})
	}

	// Sort by similarity score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Apply pagination
	total := len(results)
	end := offset + limit
	if end > total {
		end = total
	}

	var pagedResults []Document
	if offset < total {
		for _, res := range results[offset:end] {
			pagedResults = append(pagedResults, res.doc)
		}
	}

	return &SearchResult{
		Documents: pagedResults,
		Total:     total,
		HasMore:   end < total,
	}, nil
}

// GetDocuments retrieves documents by their IDs
func (s *InMemoryStore) GetDocuments(_ context.Context, ids []string) ([]Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isOpen {
		return nil, errRegistry.New(ErrCodeStoreFailure).WithDetail("reason", "store is closed")
	}

	if len(ids) == 0 {
		return nil, errRegistry.New(ErrCodeInvalidParameter).WithDetail("reason", "empty id list")
	}

	docs := make([]Document, 0, len(ids))
	for _, id := range ids {
		if doc, exists := s.docs[id]; exists {
			docs = append(docs, doc)
		}
	}

	return docs, nil
}

// DeleteDocuments removes documents by their IDs
func (s *InMemoryStore) DeleteDocuments(_ context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isOpen {
		return errRegistry.New(ErrCodeStoreFailure).WithDetail("reason", "store is closed")
	}

	if len(ids) == 0 {
		return errRegistry.New(ErrCodeInvalidParameter).WithDetail("reason", "empty id list")
	}

	for _, id := range ids {
		delete(s.docs, id)
		delete(s.vectors, id)
	}

	return nil
}

// DeleteByFilter removes documents matching the filter
func (s *InMemoryStore) DeleteByFilter(_ context.Context, filter Filter) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isOpen {
		return errRegistry.New(ErrCodeStoreFailure).WithDetail("reason", "store is closed")
	}

	if filter == nil {
		return errRegistry.New(ErrCodeInvalidFilter).WithDetail("reason", "nil filter")
	}

	var idsToDelete []string
	for id, doc := range s.docs {
		if documentMatchesFilter(doc, filter) {
			idsToDelete = append(idsToDelete, id)
		}
	}

	for _, id := range idsToDelete {
		delete(s.docs, id)
		delete(s.vectors, id)
	}

	return nil
}

// DocumentExists checks if documents exist
func (s *InMemoryStore) DocumentExists(_ context.Context, ids []string) ([]bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isOpen {
		return nil, errRegistry.New(ErrCodeStoreFailure).WithDetail("reason", "store is closed")
	}

	if len(ids) == 0 {
		return nil, errRegistry.New(ErrCodeInvalidParameter).WithDetail("reason", "empty id list")
	}

	exists := make([]bool, len(ids))
	for i, id := range ids {
		_, exists[i] = s.docs[id]
	}

	return exists, nil
}

// Count returns the number of documents matching a filter
func (s *InMemoryStore) Count(_ context.Context, filter Filter) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isOpen {
		return 0, errRegistry.New(ErrCodeStoreFailure).WithDetail("reason", "store is closed")
	}

	// If no filter, return total count
	if filter == nil {
		return len(s.docs), nil
	}

	// Count documents matching the filter
	count := 0
	for _, doc := range s.docs {
		if documentMatchesFilter(doc, filter) {
			count++
		}
	}

	return count, nil
}

// Close the store
func (s *InMemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.isOpen = false
	return nil
}

// Features returns the supported features
func (s *InMemoryStore) Features() map[string]bool {
	return map[string]bool{
		"filter":      true,
		"delete":      true,
		"count":       true,
		"namespace":   true,
		"metadata":    true,
		"exact_match": true,
	}
}

