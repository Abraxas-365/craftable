package document

import (
	"context"
	"fmt"
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
		"extension":  strings.ToLower(filepath.Ext(path)),
		"created_at": info.ModTime,
		"size":       info.Size,
	}
}

// Load implements DocumentLoader for text files
func (l *TextLoader) Load(ctx context.Context) ([]*Document, error) {
	info, err := l.FS.Stat(ctx, l.Path)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("path", l.Path)
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
			WithDetail("path", path)
	}

	data, err := l.FS.ReadFile(ctx, path)
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
func (l *TextLoader) loadDirectory(ctx context.Context, dirPath string) ([]*Document, error) {
	var documents []*Document

	entries, err := l.FS.List(ctx, dirPath)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("directory", dirPath)
	}

	for _, entry := range entries {
		entryPath := l.FS.Join(dirPath, entry.Name)

		if entry.IsDir {
			if l.RecursiveDir {
				// Recursively process subdirectories
				subDocs, err := l.loadDirectory(ctx, entryPath)
				if err != nil {
					// Log but continue
					fmt.Printf("Error loading directory %s: %v\n", entryPath, err)
					continue
				}
				documents = append(documents, subDocs...)
			}
			// Skip directories if not recursive
			continue
		}

		// Skip non-text files
		ext := strings.ToLower(filepath.Ext(entryPath))
		if !isTextFileExtension(ext) {
			continue
		}

		docs, err := l.loadFile(ctx, entryPath)
		if err != nil {
			// Log but continue
			fmt.Printf("Error loading file %s: %v\n", entryPath, err)
			continue
		}

		documents = append(documents, docs...)
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

