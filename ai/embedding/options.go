package embedding

// EmbeddingOptions contains options for generating embeddings
type EmbeddingOptions struct {
	// Model is the embedding model to use
	Model string

	// Dimensions specifies the dimensions of the embedding vectors (if the model supports variable dimensions)
	Dimensions int

	// User is an optional user identifier for tracking and rate limiting
	User string
}

// Option is a function type to modify EmbeddingOptions
type Option func(*EmbeddingOptions)

// WithModel sets the embedding model to use
func WithModel(model string) Option {
	return func(o *EmbeddingOptions) {
		o.Model = model
	}
}

// WithDimensions sets the dimensions for the embedding vectors
func WithDimensions(dimensions int) Option {
	return func(o *EmbeddingOptions) {
		o.Dimensions = dimensions
	}
}

// WithUser sets the user identifier
func WithUser(user string) Option {
	return func(o *EmbeddingOptions) {
		o.User = user
	}
}

// DefaultOptions returns the default embedding options
func DefaultOptions() *EmbeddingOptions {
	return &EmbeddingOptions{
		// Default model will be provider-specific
		Dimensions: 0, // Default to model's default dimensions
	}
}
