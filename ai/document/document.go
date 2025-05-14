package document

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// SaveToDirectory saves all documents in the collection to a directory
func (c *Collection) SaveToDirectory(dirPath string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("directory", dirPath)
	}

	// Save collection metadata
	metadataFile := filepath.Join(dirPath, "collection.json")
	metadataBytes, err := json.MarshalIndent(map[string]interface{}{
		"name":       c.Name,
		"metadata":   c.Metadata,
		"created_at": c.CreatedAt,
		"updated_at": c.UpdatedAt,
		"count":      len(c.Documents),
	}, "", "  ")
	if err != nil {
		return errRegistry.NewWithCause(ErrCodeSerializationFail, err)
	}

	if err := os.WriteFile(metadataFile, metadataBytes, 0644); err != nil {
		return errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("file", metadataFile)
	}

	// Save individual documents
	for _, doc := range c.Documents {
		filename := fmt.Sprintf("%s.json", doc.ID)
		filePath := filepath.Join(dirPath, filename)
		if err := doc.SaveToFile(filePath); err != nil {
			return err
		}
	}

	return nil
}

// LoadFromDirectory loads a collection from a directory
func LoadFromDirectory(dirPath string) (*Collection, error) {
	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil, errRegistry.NewWithCause(ErrCodeDocumentNotFound, err).
			WithDetail("directory", dirPath)
	}

	// Load collection metadata
	metadataFile := filepath.Join(dirPath, "collection.json")
	metadataBytes, err := os.ReadFile(metadataFile)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("file", metadataFile)
	}

	var metadata struct {
		Name      string                 `json:"name"`
		Metadata  map[string]interface{} `json:"metadata"`
		CreatedAt time.Time              `json:"created_at"`
		UpdatedAt time.Time              `json:"updated_at"`
	}

	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeDeserializeFail, err)
	}

	collection := &Collection{
		Name:      metadata.Name,
		Documents: []*Document{},
		Metadata:  metadata.Metadata,
		CreatedAt: metadata.CreatedAt,
		UpdatedAt: metadata.UpdatedAt,
	}

	// Load all document files
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("directory", dirPath)
	}

	for _, file := range files {
		if file.IsDir() || file.Name() == "collection.json" {
			continue
		}

		if strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join(dirPath, file.Name())
			doc, err := LoadFromFile(filePath)
			if err != nil {
				return nil, err
			}
			collection.Documents = append(collection.Documents, doc)
		}
	}

	return collection, nil
}

// DocumentLoader defines an interface for loading documents from various sources
type DocumentLoader interface {
	// Load loads documents from a source
	Load() ([]*Document, error)
}

// TextLoader loads documents from text files
type TextLoader struct {
	// Path to load from (file or directory)
	Path string

	// RecursiveDir enables recursive directory scanning
	RecursiveDir bool

	// Encoding specifies text encoding
	Encoding string

	// MetadataExtractors are functions that generate metadata from loaded files
	MetadataExtractors []MetadataExtractor
}

// MetadataExtractor is a function that extracts metadata from file information
type MetadataExtractor func(path string, info os.FileInfo) map[string]interface{}

// NewTextLoader creates a new text file loader
func NewTextLoader(path string, recursive bool) *TextLoader {
	return &TextLoader{
		Path:         path,
		RecursiveDir: recursive,
		Encoding:     "utf-8",
		MetadataExtractors: []MetadataExtractor{
			StandardMetadataExtractor,
		},
	}
}

// StandardMetadataExtractor extracts common metadata from a file
func StandardMetadataExtractor(path string, info os.FileInfo) map[string]interface{} {
	return map[string]interface{}{
		"source":     path,
		"filename":   info.Name(),
		"extension":  strings.ToLower(filepath.Ext(path)),
		"created_at": info.ModTime(),
		"size":       info.Size(),
	}
}

// Load implements DocumentLoader for text files
func (l *TextLoader) Load() ([]*Document, error) {
	info, err := os.Stat(l.Path)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("path", l.Path)
	}

	if info.IsDir() {
		return l.loadDirectory(l.Path)
	}
	return l.loadFile(l.Path)
}

// loadFile loads a single text file
func (l *TextLoader) loadFile(path string) ([]*Document, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("path", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("path", path)
	}

	metadata := make(map[string]interface{})
	for _, extractor := range l.MetadataExtractors {
		for k, v := range extractor(path, info) {
			metadata[k] = v
		}
	}

	doc := NewDocumentWithMetadata(string(data), metadata)
	doc.Source = path
	return []*Document{doc}, nil
}

// loadDirectory loads text files from a directory
func (l *TextLoader) loadDirectory(dirPath string) ([]*Document, error) {
	var documents []*Document

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories unless we're scanning recursively
		if info.IsDir() {
			if path == dirPath || l.RecursiveDir {
				return nil // Continue walking
			}
			return filepath.SkipDir
		}

		// Skip non-text files (basic check, could be enhanced)
		ext := strings.ToLower(filepath.Ext(path))
		if !isTextFileExtension(ext) {
			return nil
		}

		docs, err := l.loadFile(path)
		if err != nil {
			// Log but continue
			fmt.Printf("Error loading file %s: %v\n", path, err)
			return nil
		}

		documents = append(documents, docs...)
		return nil
	}

	if err := filepath.Walk(dirPath, walkFn); err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("directory", dirPath)
	}

	return documents, nil
}

// isTextFileExtension checks if a file extension is likely to be a text file
func isTextFileExtension(ext string) bool {
	textExtensions := map[string]bool{
		".txt":  true,
		".md":   true,
		".csv":  true,
		".json": true,
		".xml":  true,
		".html": true,
		".htm":  true,
		".log":  true,
		".go":   true,
		".py":   true,
		".js":   true,
		".ts":   true,
		".c":    true,
		".cpp":  true,
		".h":    true,
		".java": true,
	}
	return textExtensions[ext]
}

// CSVLoader loads documents from CSV files
type CSVLoader struct {
	// Path to the CSV file
	Path string

	// Delimiter character
	Delimiter rune

	// HasHeader indicates if CSV has header row
	HasHeader bool
}

// NewCSVLoader creates a new CSV loader
func NewCSVLoader(path string) *CSVLoader {
	return &CSVLoader{
		Path:      path,
		Delimiter: ',',
		HasHeader: true,
	}
}

// Load implements DocumentLoader for CSV files
// Returns one document per row with column values as metadata
func (l *CSVLoader) Load() ([]*Document, error) {
	// Implementation details omitted for brevity
	// This would parse CSV file and create documents with metadata for columns
	return nil, errRegistry.New(ErrCodeUnsupported).
		WithDetail("reason", "CSV loader not fully implemented")
}

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

// StreamingDocument represents a document that's being streamed from a source
type StreamingDocument struct {
	Document
	reader io.Reader
	buffer []byte
}

// NewStreamingDocument creates a streaming document from a reader
func NewStreamingDocument(reader io.Reader) *StreamingDocument {
	return &StreamingDocument{
		Document: Document{
			ID:        uuid.New().String(),
			Metadata:  make(map[string]interface{}),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   1,
		},
		reader: reader,
		buffer: make([]byte, 0, 1024),
	}
}

// ReadAll reads the entire content from the stream
func (d *StreamingDocument) ReadAll() error {
	content, err := io.ReadAll(d.reader)
	if err != nil {
		return errRegistry.NewWithCause(ErrCodeIOFailure, err)
	}
	d.PageContent = string(content)
	return nil
}

// ReadChunk reads the next chunk from the stream
func (d *StreamingDocument) ReadChunk(size int) (bool, error) {
	buffer := make([]byte, size)
	n, err := d.reader.Read(buffer)

	if err == io.EOF {
		return true, nil
	}

	if err != nil {
		return false, errRegistry.NewWithCause(ErrCodeIOFailure, err)
	}

	d.buffer = append(d.buffer, buffer[:n]...)
	d.PageContent = string(d.buffer)
	return false, nil
}

// Error codes
const ErrCodeUnsupported = errx.Code("UNSUPPORTED_OPERATION")
