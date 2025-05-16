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

// FileReader provides read-only operations
type FileReader interface {
	ReadFile(ctx context.Context, path string) ([]byte, error)
	ReadFileStream(ctx context.Context, path string) (io.ReadCloser, error)
	Stat(ctx context.Context, path string) (FileInfo, error)
	List(ctx context.Context, path string) ([]FileInfo, error)
	Exists(ctx context.Context, path string) (bool, error)
}

// FileWriter provides write operations
type FileWriter interface {
	WriteFile(ctx context.Context, path string, data []byte) error
	WriteFileStream(ctx context.Context, path string, r io.Reader) error
	CreateDir(ctx context.Context, path string) error
}

// FileDeleter provides deletion operations
type FileDeleter interface {
	DeleteFile(ctx context.Context, path string) error
	DeleteDir(ctx context.Context, path string, recursive bool) error
}

// PathOperations provides path manipulation functionality
type PathOperations interface {
	Join(elem ...string) string
}

// FileSystem combines all file operations
type FileSystem interface {
	FileReader
	FileWriter
	FileDeleter
	PathOperations
}

type PathReader interface {
	FileReader
	PathOperations
}
