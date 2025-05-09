package dtox

import "runtime"

// Options holds configuration options for the mapper
type Options struct {
	// UseParallelProcessing enables parallel processing for batch operations
	UseParallelProcessing bool

	// ParallelThreshold defines the minimum batch size for parallel processing
	ParallelThreshold int

	// MaxConcurrency limits the maximum number of concurrent operations
	MaxConcurrency int

	// FieldMappingMode defines the default strategy for field mapping
	FieldMappingMode FieldMappingMode
}

// defaultOptions creates a default options configuration
func defaultOptions() *Options {
	return &Options{
		UseParallelProcessing: true,
		ParallelThreshold:     100,
		MaxConcurrency:        runtime.NumCPU(),
		FieldMappingMode:      FlexibleMatch,
	}
}

// NewOptions creates a customized options configuration
func NewOptions() *Options {
	return defaultOptions()
}

// WithParallelProcessing enables or disables parallel processing
func (o *Options) WithParallelProcessing(enable bool) *Options {
	o.UseParallelProcessing = enable
	return o
}

// WithParallelThreshold sets the minimum batch size for parallel processing
func (o *Options) WithParallelThreshold(threshold int) *Options {
	o.ParallelThreshold = threshold
	return o
}

// WithMaxConcurrency sets the maximum number of concurrent operations
func (o *Options) WithMaxConcurrency(max int) *Options {
	o.MaxConcurrency = max
	return o
}

// WithFieldMappingMode sets the default field mapping strategy
func (o *Options) WithFieldMappingMode(mode FieldMappingMode) *Options {
	o.FieldMappingMode = mode
	return o
}
