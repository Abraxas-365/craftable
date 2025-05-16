package document

import (
	"context"
	"io"
	"time"

	"github.com/Abraxas-365/craftable/fsx"
	"github.com/google/uuid"
)

// StreamingDocument represents a document that's being streamed from a source
type StreamingDocument struct {
	Document
	reader  io.ReadCloser
	buffer  []byte
	context context.Context
}

// NewStreamingDocumentFromFS creates a streaming document from a file in the filesystem
func NewStreamingDocumentFromFS(ctx context.Context, fs fsx.PathReader, path string) (*StreamingDocument, error) {
	reader, err := fs.ReadFileStream(ctx, path)
	if err != nil {
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("path", path)
	}

	info, err := fs.Stat(ctx, path)
	if err != nil {
		reader.Close() // Make sure to close the reader if we fail
		return nil, errRegistry.NewWithCause(ErrCodeIOFailure, err).
			WithDetail("path", path)
	}

	// Create basic metadata from file info
	metadata := map[string]interface{}{
		"source":     path,
		"filename":   info.Name,
		"size":       info.Size,
		"created_at": info.ModTime,
	}

	doc := &StreamingDocument{
		Document: Document{
			ID:        uuid.New().String(),
			Source:    path,
			Metadata:  metadata,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   1,
		},
		reader:  reader,
		buffer:  make([]byte, 0, 1024),
		context: ctx,
	}

	return doc, nil
}

// NewStreamingDocument creates a streaming document from a reader
func NewStreamingDocument(ctx context.Context, reader io.ReadCloser) *StreamingDocument {
	return &StreamingDocument{
		Document: Document{
			ID:        uuid.New().String(),
			Metadata:  make(map[string]interface{}),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   1,
		},
		reader:  reader,
		buffer:  make([]byte, 0, 1024),
		context: ctx,
	}
}

// ReadAll reads the entire content from the stream
func (d *StreamingDocument) ReadAll() error {
	// Check if the context has been canceled
	if err := d.context.Err(); err != nil {
		return errRegistry.NewWithCause(ErrCodeIOFailure, err)
	}

	content, err := io.ReadAll(d.reader)
	if err != nil {
		return errRegistry.NewWithCause(ErrCodeIOFailure, err)
	}
	d.PageContent = string(content)
	return nil
}

// ReadChunk reads the next chunk from the stream
func (d *StreamingDocument) ReadChunk(size int) (bool, error) {
	// Check if the context has been canceled
	if err := d.context.Err(); err != nil {
		return false, errRegistry.NewWithCause(ErrCodeIOFailure, err)
	}

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

// Close frees resources associated with the document
func (d *StreamingDocument) Close() error {
	if d.reader != nil {
		return d.reader.Close()
	}
	return nil
}

