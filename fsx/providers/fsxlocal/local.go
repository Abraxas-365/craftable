package fsxlocal

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"os"
	"path/filepath"

	"github.com/Abraxas-365/craftable/fsx"
)

// LocalFS implements the fsx.FileSystem interface using the local file system
type LocalFS struct {
	// Optional root directory to restrict operations to a subdirectory
	root string
}

// NewLocalFS creates a new local file system instance
func NewLocalFS(root string) *LocalFS {
	// Normalize root path
	if root != "" {
		root = filepath.Clean(root)
	}
	return &LocalFS{root: root}
}

// resolvePath resolves and validates a path within root directory
func (l *LocalFS) resolvePath(path string) string {
	if l.root == "" {
		return filepath.Clean(path)
	}
	return filepath.Join(l.root, path)
}

// ReadFile reads a file's content
func (l *LocalFS) ReadFile(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(l.resolvePath(path))
}

// ReadFileStream opens a file for reading
func (l *LocalFS) ReadFileStream(_ context.Context, path string) (io.ReadCloser, error) {
	return os.Open(l.resolvePath(path))
}

// Stat returns file information
func (l *LocalFS) Stat(_ context.Context, path string) (fsx.FileInfo, error) {
	info, err := os.Stat(l.resolvePath(path))
	if err != nil {
		return fsx.FileInfo{}, err
	}

	return convertOsFileInfo(info, path), nil
}

// List returns directory contents
func (l *LocalFS) List(_ context.Context, path string) ([]fsx.FileInfo, error) {
	entries, err := os.ReadDir(l.resolvePath(path))
	if err != nil {
		return nil, err
	}

	result := make([]fsx.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}

		entryPath := filepath.Join(path, entry.Name())
		result = append(result, convertOsFileInfo(info, entryPath))
	}

	return result, nil
}

// WriteFile writes data to a file
func (l *LocalFS) WriteFile(_ context.Context, path string, data []byte) error {
	fullPath := l.resolvePath(path)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, 0644)
}

// WriteFileStream writes data from a reader to a file
func (l *LocalFS) WriteFileStream(_ context.Context, path string, r io.Reader) error {
	fullPath := l.resolvePath(path)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, r)
	return err
}

// CreateDir creates a directory
func (l *LocalFS) CreateDir(_ context.Context, path string) error {
	return os.MkdirAll(l.resolvePath(path), 0755)
}

// DeleteFile deletes a file
func (l *LocalFS) DeleteFile(_ context.Context, path string) error {
	info, err := os.Stat(l.resolvePath(path))
	if err != nil {
		return err
	}

	if info.IsDir() {
		return fmt.Errorf("cannot use DeleteFile on directory: %s", path)
	}

	return os.Remove(l.resolvePath(path))
}

// DeleteDir deletes a directory
func (l *LocalFS) DeleteDir(_ context.Context, path string, recursive bool) error {
	resolvedPath := l.resolvePath(path)

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}

	if recursive {
		return os.RemoveAll(resolvedPath)
	}

	// For non-recursive delete, check if directory is empty
	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		return err
	}

	if len(entries) > 0 {
		return fmt.Errorf("directory not empty: %s", path)
	}

	return os.Remove(resolvedPath)
}

// Join joins path elements
func (l *LocalFS) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Exists checks if a file or directory exists
func (l *LocalFS) Exists(_ context.Context, path string) (bool, error) {
	_, err := os.Stat(l.resolvePath(path))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Helper function to convert os.FileInfo to our FileInfo
func convertOsFileInfo(info fs.FileInfo, path string) fsx.FileInfo {
	// Try to detect content type for files
	contentType := ""
	if !info.IsDir() {
		ext := filepath.Ext(info.Name())
		contentType = mime.TypeByExtension(ext)
	}

	return fsx.FileInfo{
		Name:        info.Name(),
		Size:        info.Size(),
		ModTime:     info.ModTime(),
		IsDir:       info.IsDir(),
		ContentType: contentType,
		Metadata:    map[string]string{}, // OS doesn't provide metadata
	}
}
