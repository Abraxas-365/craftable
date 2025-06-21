package eventx

import (
	"context"
	"reflect"
)

// EventHandler is a function that processes events
type EventHandler func(Event) error

// TypedEventHandler provides type-safe event handling
type TypedEventHandler[T any] func(TypedEvent[T]) error

// EventFilter determines if an event should be processed
type EventFilter func(Event) bool

// EventBus defines the interface for event bus implementations
type EventBus interface {
	// Connect establishes connection to the underlying message system
	Connect(ctx context.Context) error

	// Disconnect closes the connection to the underlying message system
	Disconnect(ctx context.Context) error

	// IsConnected returns true if the bus is connected
	IsConnected() bool

	// Subscribe registers an event handler for a specific event type
	Subscribe(ctx context.Context, eventType string, handler EventHandler) error

	// Unsubscribe removes handlers for a specific event type
	Unsubscribe(ctx context.Context, eventType string) error

	// Publish publishes an event to all registered handlers
	Publish(ctx context.Context, event Event) error

	// PublishBatch publishes multiple events in a single operation
	PublishBatch(ctx context.Context, events []Event) error

	// AddFilter adds a filter for a specific event type
	AddFilter(eventType string, filter EventFilter) error

	// RemoveFilter removes filters for a specific event type
	RemoveFilter(eventType string) error

	// ListEventTypes returns all registered event types
	ListEventTypes() []string

	// HandlerCount returns the number of handlers for an event type
	HandlerCount(eventType string) int

	// Health returns the health status of the event bus
	Health(ctx context.Context) error
}

// AsyncEventBus extends EventBus with async capabilities
type AsyncEventBus interface {
	EventBus

	// PublishAsync publishes an event asynchronously
	PublishAsync(ctx context.Context, event Event) error

	// PublishBatchAsync publishes multiple events asynchronously
	PublishBatchAsync(ctx context.Context, events []Event) error
}

// MetricsEventBus extends EventBus with metrics capabilities
type MetricsEventBus interface {
	EventBus

	// GetMetrics returns event bus metrics
	GetMetrics() BusMetrics
}

// BusMetrics represents event bus metrics
type BusMetrics struct {
	EventsPublished   int64 `json:"events_published"`
	EventsProcessed   int64 `json:"events_processed"`
	EventsFailed      int64 `json:"events_failed"`
	ActiveSubscribers int   `json:"active_subscribers"`
	ConnectionStatus  bool  `json:"connection_status"`
}

// BusConfig represents common configuration for event buses
type BusConfig struct {
	// Connection settings
	URL            string `json:"url"`
	ConnectionName string `json:"connection_name"`
	MaxConnections int    `json:"max_connections"`
	ConnectTimeout int    `json:"connect_timeout_seconds"`
	ReconnectDelay int    `json:"reconnect_delay_seconds"`
	MaxReconnects  int    `json:"max_reconnects"`

	// Publishing settings
	PublishTimeout    int  `json:"publish_timeout_seconds"`
	EnableCompression bool `json:"enable_compression"`
	BatchSize         int  `json:"batch_size"`

	// Subscription settings
	PrefetchCount     int  `json:"prefetch_count"`
	AutoAck           bool `json:"auto_ack"`
	ExclusiveConsumer bool `json:"exclusive_consumer"`

	// Reliability settings
	EnablePersistence bool `json:"enable_persistence"`
	EnableRetry       bool `json:"enable_retry"`
	MaxRetries        int  `json:"max_retries"`
	RetryDelay        int  `json:"retry_delay_seconds"`

	// Monitoring
	EnableMetrics bool `json:"enable_metrics"`
	EnableLogging bool `json:"enable_logging"`
}

// DefaultBusConfig returns default configuration
func DefaultBusConfig() BusConfig {
	return BusConfig{
		ConnectionName:    "eventx-bus",
		MaxConnections:    10,
		ConnectTimeout:    30,
		ReconnectDelay:    5,
		MaxReconnects:     3,
		PublishTimeout:    30,
		EnableCompression: false,
		BatchSize:         100,
		PrefetchCount:     10,
		AutoAck:           true,
		ExclusiveConsumer: false,
		EnablePersistence: true,
		EnableRetry:       true,
		MaxRetries:        3,
		RetryDelay:        1,
		EnableMetrics:     true,
		EnableLogging:     true,
	}
}

// SubscribeTyped registers a typed event handler
func SubscribeTyped[T any](bus EventBus, ctx context.Context, eventType string, handler TypedEventHandler[T]) error {
	return bus.Subscribe(ctx, eventType, func(e Event) error {
		if typedEvent, ok := e.(TypedEvent[T]); ok {
			return handler(typedEvent)
		}
		return ErrorRegistry.New(ErrInvalidEventType).
			WithDetail("expected_type", reflect.TypeOf((*T)(nil)).Elem().String()).
			WithDetail("actual_type", reflect.TypeOf(e.Payload()).String())
	})
}
