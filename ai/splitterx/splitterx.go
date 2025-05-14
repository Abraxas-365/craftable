package splitterx

import (
	"fmt"
	"strings"

	"github.com/Abraxas-365/craftable/ai/document"
	"github.com/pkoukk/tiktoken-go"
)

// Splitter defines the interface for document splitting strategies
type Splitter interface {
	// SplitText divides text into smaller chunks
	SplitText(text string) ([]string, error)

	// SplitDocuments splits multiple documents, preserving metadata
	SplitDocuments(docs []document.Document) ([]document.Document, error)

	// GetType returns the type of splitter
	GetType() string

	// GetParameters returns the splitter's configuration parameters
	GetParameters() map[string]interface{}
}

// SimpleCharacterSplitter splits text by character count
type SimpleCharacterSplitter struct {
	ChunkSize    int
	ChunkOverlap int
	Separator    string
}

// NewSimpleCharacterSplitter creates a new character-based splitter
func NewSimpleCharacterSplitter(chunkSize, chunkOverlap int) *SimpleCharacterSplitter {
	if chunkSize <= 0 {
		chunkSize = 1000
	}
	if chunkOverlap < 0 {
		chunkOverlap = 0
	}
	if chunkOverlap >= chunkSize {
		chunkOverlap = chunkSize / 2
	}

	return &SimpleCharacterSplitter{
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
		Separator:    "\n\n",
	}
}

// SplitText implements character-based text splitting
func (s *SimpleCharacterSplitter) SplitText(text string) ([]string, error) {
	fmt.Println("SimpleCharacterSplitter.SplitText called")
	if text == "" {
		return []string{}, nil
	}

	// Split text by separator
	var splits []string
	if s.Separator != "" {
		splits = strings.Split(text, s.Separator)
	} else {
		splits = []string{text}
	}

	// Merge splits into chunks of appropriate size
	var chunks []string
	var currentChunk strings.Builder
	currentLength := 0

	for _, split := range splits {
		splitLength := len(split)

		// Skip empty splits
		if splitLength == 0 {
			continue
		}

		// If a single split is larger than chunk size, we need to divide it
		if splitLength > s.ChunkSize {
			// If there's content in the current chunk, add it to results
			if currentLength > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
				currentLength = 0
			}

			// Divide the large split into multiple chunks
			for i := 0; i < splitLength; i += s.ChunkSize - s.ChunkOverlap {
				end := i + s.ChunkSize
				if end > splitLength {
					end = splitLength
				}
				chunks = append(chunks, split[i:end])
			}
			continue
		}

		// If adding this split would exceed chunk size, finalize current chunk
		if currentLength+splitLength > s.ChunkSize {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentLength = 0
		}

		// Add the split to the current chunk
		if currentLength > 0 && splitLength > 0 {
			// Add a space between chunks if needed
			currentChunk.WriteString(" ")
			currentLength++
		}
		currentChunk.WriteString(split)
		currentLength += splitLength
	}

	// Add the final chunk if there's anything left
	if currentLength > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	fmt.Printf("SimpleCharacterSplitter generated %d chunks\n", len(chunks))
	return chunks, nil
}

// SplitDocuments splits multiple documents, preserving metadata
func (s *SimpleCharacterSplitter) SplitDocuments(docs []document.Document) ([]document.Document, error) {
	fmt.Println("SimpleCharacterSplitter.SplitDocuments called")
	var result []document.Document

	for i, doc := range docs {
		chunks, err := s.SplitText(doc.PageContent)
		if err != nil {
			return nil, fmt.Errorf("failed to split document at index %d: %w", i, err)
		}

		for j, chunk := range chunks {
			// Create a copy of the original metadata
			metadata := make(map[string]interface{}, len(doc.Metadata))
			for k, v := range doc.Metadata {
				metadata[k] = v
			}

			// Add chunk information to metadata
			metadata["chunk_index"] = j
			metadata["chunk_total"] = len(chunks)
			metadata["splitter_type"] = "character"

			// Create ID based on original document if it exists
			var docID string
			if doc.ID != "" {
				docID = fmt.Sprintf("%s_chunk_%d", doc.ID, j)
			} else {
				docID = fmt.Sprintf("doc_%d_chunk_%d", i, j)
			}

			result = append(result, document.Document{
				ID:          docID,
				PageContent: chunk,
				Metadata:    metadata,
			})
		}
	}

	return result, nil
}

// GetType returns the type of splitter
func (s *SimpleCharacterSplitter) GetType() string {
	return "character"
}

// GetParameters returns the splitter's configuration parameters
func (s *SimpleCharacterSplitter) GetParameters() map[string]interface{} {
	return map[string]interface{}{
		"chunk_size":    s.ChunkSize,
		"chunk_overlap": s.ChunkOverlap,
		"separator":     s.Separator,
	}
}

// WithSeparator sets the separator
func (s *SimpleCharacterSplitter) WithSeparator(separator string) *SimpleCharacterSplitter {
	s.Separator = separator
	return s
}

// SimpleTokenSplitter splits text based on token count using a tokenizer
type SimpleTokenSplitter struct {
	ChunkSize    int
	ChunkOverlap int
	Model        string
	tokenizer    *tiktoken.Tiktoken
}

// NewSimpleTokenSplitter creates a new token-based splitter
func NewSimpleTokenSplitter(chunkSize, chunkOverlap int, model string) (*SimpleTokenSplitter, error) {
	if chunkSize <= 0 {
		chunkSize = 512
	}
	if chunkOverlap < 0 {
		chunkOverlap = 0
	}
	if chunkOverlap >= chunkSize {
		chunkOverlap = chunkSize / 2
	}
	if model == "" {
		model = "text-embedding-3-small"
	}

	// Get the appropriate encoding for the model
	encodingName := getEncodingForModel(model)
	encoding, err := tiktoken.GetEncoding(encodingName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tokenizer for model %s: %w", model, err)
	}

	return &SimpleTokenSplitter{
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
		Model:        model,
		tokenizer:    encoding,
	}, nil
}

// getEncodingForModel returns the appropriate encoding name for a given model
func getEncodingForModel(model string) string {
	// GPT-4 models
	if strings.HasPrefix(model, "gpt-4") {
		return "cl100k_base"
	}

	// GPT-3.5 and embedding models
	if strings.HasPrefix(model, "gpt-3.5-turbo") ||
		model == "text-embedding-ada-002" ||
		model == "text-embedding-3-small" ||
		model == "text-embedding-3-large" {
		return "cl100k_base"
	}

	// Default to cl100k_base if model is unknown
	return "cl100k_base"
}

// SplitText implements token-based text splitting
func (s *SimpleTokenSplitter) SplitText(text string) ([]string, error) {
	fmt.Println("SimpleTokenSplitter.SplitText called")
	if text == "" {
		return []string{}, nil
	}

	// Encode the text to tokens
	tokens := s.tokenizer.Encode(text, nil, nil)

	// Split tokens into chunks
	var chunks []string

	// Split tokens into chunks based on chunk size and overlap
	for i := 0; i < len(tokens); i += s.ChunkSize - s.ChunkOverlap {
		end := i + s.ChunkSize
		if end > len(tokens) {
			end = len(tokens)
		}

		chunkTokens := tokens[i:end]
		chunk := s.tokenizer.Decode(chunkTokens)

		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		// Last chunk, no need to continue
		if end == len(tokens) {
			break
		}
	}

	fmt.Printf("SimpleTokenSplitter generated %d chunks\n", len(chunks))
	return chunks, nil
}

// SplitDocuments splits multiple documents, preserving metadata
func (s *SimpleTokenSplitter) SplitDocuments(docs []document.Document) ([]document.Document, error) {
	fmt.Println("SimpleTokenSplitter.SplitDocuments called")
	var result []document.Document

	for i, doc := range docs {
		chunks, err := s.SplitText(doc.PageContent)
		if err != nil {
			return nil, fmt.Errorf("failed to split document at index %d: %w", i, err)
		}

		for j, chunk := range chunks {
			// Create a copy of the original metadata
			metadata := make(map[string]interface{}, len(doc.Metadata))
			for k, v := range doc.Metadata {
				metadata[k] = v
			}

			// Add chunk information to metadata
			metadata["chunk_index"] = j
			metadata["chunk_total"] = len(chunks)
			metadata["splitter_type"] = "token"
			metadata["model"] = s.Model

			// Create ID based on original document if it exists
			var docID string
			if doc.ID != "" {
				docID = fmt.Sprintf("%s_chunk_%d", doc.ID, j)
			} else {
				docID = fmt.Sprintf("doc_%d_chunk_%d", i, j)
			}

			result = append(result, document.Document{
				ID:          docID,
				PageContent: chunk,
				Metadata:    metadata,
			})
		}
	}

	return result, nil
}

// GetType returns the type of splitter
func (s *SimpleTokenSplitter) GetType() string {
	return "token"
}

// GetParameters returns the splitter's configuration parameters
func (s *SimpleTokenSplitter) GetParameters() map[string]interface{} {
	return map[string]interface{}{
		"chunk_size":    s.ChunkSize,
		"chunk_overlap": s.ChunkOverlap,
		"model":         s.Model,
	}
}

// CountTokens counts the number of tokens in a text
func (s *SimpleTokenSplitter) CountTokens(text string) int {
	if text == "" {
		return 0
	}
	tokens := s.tokenizer.Encode(text, nil, nil)
	return len(tokens)
}

