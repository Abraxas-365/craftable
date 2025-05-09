package ocr

// OCROptions contains options for OCR operations
type OCROptions struct {
	// Model is the OCR model to use
	Model string

	// Language specifies the expected language(s) in the image
	Language string

	// DetectOrientation automatically rotates the image if needed
	DetectOrientation bool

	// DetailsLevel controls how much detail is returned
	// "high" returns bounding boxes and confidence scores
	// "medium" returns confidence scores only
	// "low" returns just the extracted text
	DetailsLevel string

	// User is an optional user identifier for tracking and rate limiting
	User string
}

// Option is a function type to modify OCROptions
type Option func(*OCROptions)

// WithModel sets the OCR model to use
func WithModel(model string) Option {
	return func(o *OCROptions) {
		o.Model = model
	}
}

// WithLanguage sets the expected language(s)
func WithLanguage(language string) Option {
	return func(o *OCROptions) {
		o.Language = language
	}
}

// WithDetectOrientation enables automatic image orientation detection
func WithDetectOrientation(detect bool) Option {
	return func(o *OCROptions) {
		o.DetectOrientation = detect
	}
}

// WithDetailsLevel sets the level of detail in the results
func WithDetailsLevel(level string) Option {
	return func(o *OCROptions) {
		o.DetailsLevel = level
	}
}

// WithUser sets the user identifier
func WithUser(user string) Option {
	return func(o *OCROptions) {
		o.User = user
	}
}

// DefaultOptions returns the default OCR options
func DefaultOptions() *OCROptions {
	return &OCROptions{
		Model:             "gpt-4-vision-preview", // Default model
		Language:          "auto",
		DetectOrientation: true,
		DetailsLevel:      "medium",
	}
}

