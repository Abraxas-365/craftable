package document

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/google/uuid"
)

// Error registry for document-specific errors
var (
	errRegistry = errx.NewRegistry("DOC")

	// Error codes
	ErrCodeInvalidDocument   = errRegistry.Register("INVALID_DOCUMENT", errx.TypeValidation, 400, "Invalid document format")
	ErrCodeDocumentNotFound  = errRegistry.Register("DOCUMENT_NOT_FOUND", errx.TypeNotFound, 404, "Document not found")
	ErrCodeSerializationFail = errRegistry.Register("SERIALIZATION_FAIL", errx.TypeSystem, 500, "Failed to serialize document")
	ErrCodeDeserializeFail   = errRegistry.Register("DESERIALIZE_FAIL", errx.TypeSystem, 500, "Failed to deserialize document")
	ErrCodeIOFailure         = errRegistry.Register("IO_FAILURE", errx.TypeSystem, 500, "I/O operation failed")
	ErrCodeInvalidFormat     = errRegistry.Register("INVALID_FORMAT", errx.TypeValidation, 400, "Invalid document format")
)

// Document represents a text document with rich metadata and versioning
type Document struct {
	// ID uniquely identifies the document
	ID string `json:"id,omitempty"`

	// PageContent contains the actual text content
	PageContent string `json:"page_content"`

	// Metadata stores document metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Source tracks where the document came from
	Source string `json:"source,omitempty"`

	// CreatedAt records when the document was created
	CreatedAt time.Time `json:"created_at,omitempty"`

	// UpdatedAt records when the document was last updated
	UpdatedAt time.Time `json:"updated_at,omitempty"`

	// Version tracks document version number
	Version int `json:"version,omitempty"`
}

// NewDocument creates a document with initialized fields and a new UUID
func NewDocument(content string) *Document {
	now := time.Now()
	return &Document{
		ID:          uuid.New().String(),
		PageContent: content,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     1,
	}
}

// NewDocumentWithMetadata creates a document with specified content and metadata
func NewDocumentWithMetadata(content string, metadata map[string]interface{}) *Document {
	doc := NewDocument(content)
	for k, v := range metadata {
		doc.Metadata[k] = v
	}
	return doc
}

// Clone creates a deep copy of a document
func (d *Document) Clone() *Document {
	clonedMetadata := make(map[string]interface{})
	for k, v := range d.Metadata {
		clonedMetadata[k] = v
	}

	return &Document{
		ID:          d.ID,
		PageContent: d.PageContent,
		Metadata:    clonedMetadata,
		Source:      d.Source,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
		Version:     d.Version,
	}
}

// AddMetadata adds or updates metadata fields
func (d *Document) AddMetadata(metadata map[string]interface{}) *Document {
	for k, v := range metadata {
		d.Metadata[k] = v
	}
	d.UpdatedAt = time.Now()
	return d
}

// DeleteMetadata removes metadata fields
func (d *Document) DeleteMetadata(keys ...string) *Document {
	for _, key := range keys {
		delete(d.Metadata, key)
	}
	d.UpdatedAt = time.Now()
	return d
}

// UpdateContent updates document content and increases version
func (d *Document) UpdateContent(content string) *Document {
	d.PageContent = content
	d.UpdatedAt = time.Now()
	d.Version++
	return d
}

// Validate checks if a document is valid
func (d *Document) Validate() error {
	if d.PageContent == "" {
		return errRegistry.New(ErrCodeInvalidDocument).
			WithDetail("reason", "empty page content")
	}
	return nil
}

// MarshalJSON provides custom JSON serialization
func (d *Document) MarshalJSON() ([]byte, error) {
	type Alias Document
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(d),
	})
}

// ToJSON serializes the document to JSON
func (d *Document) ToJSON() ([]byte, error) {
	data, err := json.Marshal(d)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeSerializationFail, err).
			WithDetail("document_id", d.ID)
	}
	return data, nil
}

// DocumentFromJSON deserializes a document from JSON
func DocumentFromJSON(data []byte) (*Document, error) {
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeDeserializeFail, err)
	}
	return &doc, nil
}

// SaveToFile saves the document to a file
func (d *Document) SaveToFile(filePath string) error {
	data, err := d.ToJSON()
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("file_path", filePath)
	}
	return nil
}

// LoadFromFile loads a document from a file
func LoadFromFile(filePath string) (*Document, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("file_path", filePath)
	}

	return DocumentFromJSON(data)
}

// Collection represents a collection of documents
type Collection struct {
	// Name of the collection
	Name string `json:"name"`

	// Documents in the collection
	Documents []*Document `json:"documents"`

	// Metadata for the collection
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// CreatedAt records when the collection was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt records when the collection was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// NewCollection creates a new document collection
func NewCollection(name string) *Collection {
	now := time.Now()
	return &Collection{
		Name:      name,
		Documents: []*Document{},
		Metadata:  make(map[string]interface{}),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddDocument adds a document to the collection
func (c *Collection) AddDocument(doc *Document) {
	c.Documents = append(c.Documents, doc)
	c.UpdatedAt = time.Now()
}

// AddDocuments adds multiple documents to the collection
func (c *Collection) AddDocuments(docs []*Document) {
	for _, doc := range docs {
		c.Documents = append(c.Documents, doc)
	}
	c.UpdatedAt = time.Now()
}

// RemoveDocument removes a document from the collection by ID
func (c *Collection) RemoveDocument(id string) bool {
	for i, doc := range c.Documents {
		if doc.ID == id {
			// Remove document at index i
			c.Documents = append(c.Documents[:i], c.Documents[i+1:]...)
			c.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// GetDocument retrieves a document by ID
func (c *Collection) GetDocument(id string) (*Document, error) {
	for _, doc := range c.Documents {
		if doc.ID == id {
			return doc, nil
		}
	}
	return nil, errRegistry.New(ErrCodeDocumentNotFound).
		WithDetail("document_id", id)
}

// Filter filters documents in the collection based on a predicate
func (c *Collection) Filter(predicate func(*Document) bool) []*Document {
	var result []*Document
	for _, doc := range c.Documents {
		if predicate(doc) {
			result = append(result, doc)
		}
	}
	return result
}

// FilterByMetadata filters documents by matching metadata values
func (c *Collection) FilterByMetadata(key string, value interface{}) []*Document {
	return c.Filter(func(doc *Document) bool {
		docValue, exists := doc.Metadata[key]
		if !exists {
			return false
		}
		return docValue == value
	})
}

// DocumentLoader defines an interface for loading documents from various sources
type DocumentLoader interface {
	// Load loads documents from a source
	Load() ([]*Document, error)
	//TODO: add strems loader
}

// TextLoader loads documents from text files

// Additional loader types could be implemented:
// - JSONLoader for JSON files/APIs
// - PDFLoader for PDF documents
// - WebLoader for web pages
// - DatabaseLoader for database records

// Transformer modifies documents in some way
type Transformer interface {
	// Transform applies the transformation to a document
	Transform(doc *Document) (*Document, error)
}

// ChainTransformer chains multiple transformers
type ChainTransformer struct {
	transformers []Transformer
}

// NewChainTransformer creates a new chain of transformers
func NewChainTransformer(transformers ...Transformer) *ChainTransformer {
	return &ChainTransformer{
		transformers: transformers,
	}
}

// Transform applies all transformers in sequence
func (t *ChainTransformer) Transform(doc *Document) (*Document, error) {
	result := doc
	var err error
	for _, transformer := range t.transformers {
		result, err = transformer.Transform(result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// Basic text transformers
type (
	// LowercaseTransformer converts text to lowercase
	LowercaseTransformer struct{}

	// TrimSpaceTransformer removes leading/trailing whitespace
	TrimSpaceTransformer struct{}

	// ReplaceTransformer replaces occurrences of a string
	ReplaceTransformer struct {
		Old string
		New string
	}
)

// Transform implements Transformer for LowercaseTransformer
func (t *LowercaseTransformer) Transform(doc *Document) (*Document, error) {
	result := doc.Clone()
	result.PageContent = strings.ToLower(result.PageContent)
	return result, nil
}

// Transform implements Transformer for TrimSpaceTransformer
func (t *TrimSpaceTransformer) Transform(doc *Document) (*Document, error) {
	result := doc.Clone()
	result.PageContent = strings.TrimSpace(result.PageContent)
	return result, nil
}

// Transform implements Transformer for ReplaceTransformer
func (t *ReplaceTransformer) Transform(doc *Document) (*Document, error) {
	result := doc.Clone()
	result.PageContent = strings.ReplaceAll(result.PageContent, t.Old, t.New)
	return result, nil
}

// Error codes
const ErrCodeUnsupported = errx.Code("UNSUPPORTED_OPERATION")
