package eventx

import (
	"time"

	"github.com/google/uuid"
)

// Event is the base interface for all events
type Event interface {
	ID() string
	Type() string
	Timestamp() time.Time
	Source() string
	Version() string
	Payload() any
	Metadata() map[string]any
}

// TypedEvent provides type-safe access to event data
type TypedEvent[T any] interface {
	Event
	Data() T
}

// EventOptions configure event creation
type EventOptions struct {
	Source   string
	Version  string
	Metadata map[string]any
}

// DefaultEventOptions returns default options
func DefaultEventOptions() EventOptions {
	return EventOptions{
		Source:   "unknown",
		Version:  "1.0",
		Metadata: make(map[string]any),
	}
}

// BaseEvent implements the Event interface with generic data support
type BaseEvent[T any] struct {
	id        string
	eventType string
	timestamp time.Time
	source    string
	version   string
	data      T
	metadata  map[string]any
}

// NewEvent creates a new typed event
func NewEvent[T any](eventType string, data T, opts ...EventOptions) TypedEvent[T] {
	options := DefaultEventOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	if options.Metadata == nil {
		options.Metadata = make(map[string]any)
	}

	return &BaseEvent[T]{
		id:        generateID(),
		eventType: eventType,
		timestamp: time.Now(),
		source:    options.Source,
		version:   options.Version,
		data:      data,
		metadata:  options.Metadata,
	}
}

// NewEventWithID creates a new event with a specific ID
func NewEventWithID[T any](id, eventType string, data T, timestamp time.Time, opts ...EventOptions) TypedEvent[T] {
	options := DefaultEventOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	if options.Metadata == nil {
		options.Metadata = make(map[string]any)
	}

	return &BaseEvent[T]{
		id:        id,
		eventType: eventType,
		timestamp: timestamp,
		source:    options.Source,
		version:   options.Version,
		data:      data,
		metadata:  options.Metadata,
	}
}

// Event interface implementation
func (e *BaseEvent[T]) ID() string               { return e.id }
func (e *BaseEvent[T]) Type() string             { return e.eventType }
func (e *BaseEvent[T]) Timestamp() time.Time     { return e.timestamp }
func (e *BaseEvent[T]) Source() string           { return e.source }
func (e *BaseEvent[T]) Version() string          { return e.version }
func (e *BaseEvent[T]) Payload() any             { return e.data }
func (e *BaseEvent[T]) Metadata() map[string]any { return e.metadata }

// TypedEvent interface implementation
func (e *BaseEvent[T]) Data() T { return e.data }

// SetMetadata adds or updates metadata
func (e *BaseEvent[T]) SetMetadata(key string, value any) {
	e.metadata[key] = value
}

// GetMetadata retrieves metadata value
func (e *BaseEvent[T]) GetMetadata(key string) (any, bool) {
	value, exists := e.metadata[key]
	return value, exists
}

// generateID creates a UUID for the event
func generateID() string {
	return uuid.New().String()
}
