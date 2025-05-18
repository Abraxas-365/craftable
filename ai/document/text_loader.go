package document

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/Abraxas-365/craftable/fsx"
)

type TextLoader struct {
	// FS is the file system interface to use
	FS fsx.PathReader

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
type MetadataExtractor func(path string, info fsx.FileInfo) map[string]interface{}

// NewTextLoader creates a new text file loader
func NewTextLoader(fs fsx.PathReader, path string, recursive bool) *TextLoader {
	return &TextLoader{
		FS:           fs,
		Path:         path,
		RecursiveDir: recursive,
		Encoding:     "utf-8",
		MetadataExtractors: []MetadataExtractor{
			StandardMetadataExtractor,
		},
	}
}

// StandardMetadataExtractor extracts common metadata from a file
func StandardMetadataExtractor(path string, info fsx.FileInfo) map[string]interface{} {
	return map[string]interface{}{
		"source":     path,
		"filename":   info.Name,
		"extension":  strings.ToLower(filepath.Ext(info.Name)),
		"created_at": info.ModTime,
		"size":       info.Size,
	}
}

// Load implements DocumentLoader for text files
func (l *TextLoader) Load(ctx context.Context) ([]*Document, error) {

	// Check if path exists
	exists, err := l.FS.Exists(ctx, l.Path)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("path", l.Path).
			WithDetail("operation", "checking existence")
	}

	if !exists {
		return nil, errRegistry.New(ErrCodeIOFailure).
			WithDetail("path", l.Path).
			WithDetail("reason", "path does not exist")
	}

	info, err := l.FS.Stat(ctx, l.Path)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("path", l.Path).
			WithDetail("operation", "stat")
	}

	if info.IsDir {
		return l.loadDirectory(ctx, l.Path)
	}

	return l.loadFile(ctx, l.Path)
}

// loadFile loads a single text file
func (l *TextLoader) loadFile(ctx context.Context, path string) ([]*Document, error) {

	info, err := l.FS.Stat(ctx, path)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("path", path).
			WithDetail("operation", "stat")
	}

	// Skip non-text files
	ext := strings.ToLower(filepath.Ext(path))
	if !isTextFileExtension(ext) {
		return []*Document{}, nil // Return empty slice, not an error
	}

	data, err := l.FS.ReadFile(ctx, path)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("path", path).
			WithDetail("operation", "read")
	}

	// Extract metadata
	metadata := make(map[string]interface{})
	for _, extractor := range l.MetadataExtractors {
		extractedMeta := extractor(path, info)
		for k, v := range extractedMeta {
			metadata[k] = v
		}
	}

	// Create document
	doc := NewDocumentWithMetadata(string(data), metadata)
	doc.Source = path

	return []*Document{doc}, nil
}

// loadDirectory loads text files from a directory
func (l *TextLoader) loadDirectory(ctx context.Context, dirPath string) ([]*Document, error) {

	var documents []*Document

	entries, err := l.FS.List(ctx, dirPath)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("directory", dirPath).
			WithDetail("operation", "list")
	}

	for _, entry := range entries {
		// Skip entries without names (shouldn't happen, but just in case)
		if entry.Name == "" {
			continue
		}

		// Skip . and .. entries if present
		if entry.Name == "." || entry.Name == ".." {
			continue
		}

		// Use the filesystem's Join method to properly construct paths
		entryPath := l.FS.Join(dirPath, entry.Name)

		if entry.IsDir {
			if l.RecursiveDir {
				// Recursively process subdirectories
				subDocs, err := l.loadDirectory(ctx, entryPath)
				if err != nil {
					// Log but continue
					continue
				}

				// Only append if we found documents
				if len(subDocs) > 0 {
					documents = append(documents, subDocs...)
				}
			} else {
			}
			continue
		}

		// Process file
		docs, err := l.loadFile(ctx, entryPath)
		if err != nil {
			// Log but continue
			continue
		}

		// Only append if we actually got documents
		if len(docs) > 0 {
			documents = append(documents, docs...)
		}
	}

	return documents, nil
}

// isTextFileExtension checks if a file extension is likely to be a text file
func isTextFileExtension(ext string) bool {
	// Remove leading dot if present
	if strings.HasPrefix(ext, ".") {
		ext = ext[1:]
	}

	textExtensions := map[string]bool{
		"txt":  true,
		"md":   true,
		"csv":  true,
		"json": true,
		"xml":  true,
		"html": true,
		"htm":  true,
		"log":  true,
		"go":   true,
		"py":   true,
		"js":   true,
		"ts":   true,
		"c":    true,
		"cpp":  true,
		"h":    true,
		"java": true,
	}

	return textExtensions[strings.ToLower(ext)]
}
