package fsx

import (
	"context"
	"io"
	"time"
)

// FileInfo represents information about a file
type FileInfo struct {
	Name        string            // Base name of the file
	Size        int64             // File size in bytes
	ModTime     time.Time         // Modification time
	IsDir       bool              // Is a directory
	ContentType string            // MIME type (when available)
	Metadata    map[string]string // Additional metadata
}

// FileSystem defines the interface for file operations
type FileSystem interface {
	// Read operations
	ReadFile(ctx context.Context, path string) ([]byte, error)
	ReadFileStream(ctx context.Context, path string) (io.ReadCloser, error)
	Stat(ctx context.Context, path string) (FileInfo, error)
	List(ctx context.Context, path string) ([]FileInfo, error)

	// Write operations
	WriteFile(ctx context.Context, path string, data []byte) error
	WriteFileStream(ctx context.Context, path string, r io.Reader) error
	CreateDir(ctx context.Context, path string) error

	// Delete operations
	DeleteFile(ctx context.Context, path string) error
	DeleteDir(ctx context.Context, path string, recursive bool) error

	// Path operations
	Join(elem ...string) string
	Exists(ctx context.Context, path string) (bool, error)
}
