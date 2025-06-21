package eventxmemory

import (
	"context"
	"sync"

	"github.com/Abraxas-365/craftable/eventx"
	"github.com/Abraxas-365/craftable/logx"
)

// MemoryBus implements EventBus interface using in-memory storage
type MemoryBus struct {
	handlers map[string][]eventx.EventHandler
	filters  map[string][]eventx.EventFilter
	metrics  eventx.BusMetrics
	mutex    sync.RWMutex
	config   eventx.BusConfig
}

// New creates a new in-memory event bus
func New(config ...eventx.BusConfig) eventx.EventBus {
	cfg := eventx.DefaultBusConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &MemoryBus{
		handlers: make(map[string][]eventx.EventHandler),
		filters:  make(map[string][]eventx.EventFilter),
		metrics:  eventx.BusMetrics{ConnectionStatus: true},
		config:   cfg,
	}
}

// Connect establishes connection (no-op for memory bus)
func (mb *MemoryBus) Connect(ctx context.Context) error {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()
	mb.metrics.ConnectionStatus = true
	return nil
}

// Disconnect closes connection (no-op for memory bus)
func (mb *MemoryBus) Disconnect(ctx context.Context) error {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()
	mb.metrics.ConnectionStatus = false
	return nil
}

// IsConnected returns connection status
func (mb *MemoryBus) IsConnected() bool {
	mb.mutex.RLock()
	defer mb.mutex.RUnlock()
	return mb.metrics.ConnectionStatus
}

// Subscribe registers an event handler
func (mb *MemoryBus) Subscribe(ctx context.Context, eventType string, handler eventx.EventHandler) error {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()

	if !mb.metrics.ConnectionStatus {
		return eventx.ErrorRegistry.New(eventx.ErrBusNotConnected)
	}

	mb.handlers[eventType] = append(mb.handlers[eventType], handler)
	mb.metrics.ActiveSubscribers++

	if mb.config.EnableLogging {
		logx.Debug("Subscribed to event type: %s, total handlers: %d", eventType, len(mb.handlers[eventType]))
	}

	return nil
}

// Unsubscribe removes handlers for an event type
func (mb *MemoryBus) Unsubscribe(ctx context.Context, eventType string) error {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()

	if handlers, exists := mb.handlers[eventType]; exists {
		mb.metrics.ActiveSubscribers -= len(handlers)
		delete(mb.handlers, eventType)
		delete(mb.filters, eventType)

		if mb.config.EnableLogging {
			logx.Debug("Unsubscribed from event type: %s, removed %d handlers", eventType, len(handlers))
		}
	}

	return nil
}

// Publish publishes an event
func (mb *MemoryBus) Publish(ctx context.Context, event eventx.Event) error {
	mb.mutex.RLock()
	handlers := make([]eventx.EventHandler, len(mb.handlers[event.Type()]))
	copy(handlers, mb.handlers[event.Type()])

	filters := make([]eventx.EventFilter, len(mb.filters[event.Type()]))
	copy(filters, mb.filters[event.Type()])
	mb.mutex.RUnlock()

	if !mb.metrics.ConnectionStatus {
		return eventx.ErrorRegistry.New(eventx.ErrBusNotConnected)
	}

	// Apply filters
	for _, filter := range filters {
		if !filter(event) {
			if mb.config.EnableLogging {
				logx.Debug("Event %s filtered out", event.ID())
			}
			return nil
		}
	}

	// Execute handlers
	var lastErr error
	for _, handler := range handlers {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := handler(event); err != nil {
				mb.mutex.Lock()
				mb.metrics.EventsFailed++
				mb.mutex.Unlock()

				if mb.config.EnableLogging {
					logx.Error("Error handling event %s: %v", event.ID(), err)
				}
				lastErr = err
			} else {
				mb.mutex.Lock()
				mb.metrics.EventsProcessed++
				mb.mutex.Unlock()
			}
		}
	}

	mb.mutex.Lock()
	mb.metrics.EventsPublished++
	mb.mutex.Unlock()

	return lastErr
}

// PublishBatch publishes multiple events
func (mb *MemoryBus) PublishBatch(ctx context.Context, events []eventx.Event) error {
	var lastErr error
	for _, event := range events {
		if err := mb.Publish(ctx, event); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// AddFilter adds a filter for an event type
func (mb *MemoryBus) AddFilter(eventType string, filter eventx.EventFilter) error {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()
	mb.filters[eventType] = append(mb.filters[eventType], filter)
	return nil
}

// RemoveFilter removes filters for an event type
func (mb *MemoryBus) RemoveFilter(eventType string) error {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()
	delete(mb.filters, eventType)
	return nil
}

// ListEventTypes returns all registered event types
func (mb *MemoryBus) ListEventTypes() []string {
	mb.mutex.RLock()
	defer mb.mutex.RUnlock()

	types := make([]string, 0, len(mb.handlers))
	for eventType := range mb.handlers {
		types = append(types, eventType)
	}
	return types
}

// HandlerCount returns the number of handlers for an event type
func (mb *MemoryBus) HandlerCount(eventType string) int {
	mb.mutex.RLock()
	defer mb.mutex.RUnlock()
	return len(mb.handlers[eventType])
}

// Health returns the health status
func (mb *MemoryBus) Health(ctx context.Context) error {
	if !mb.IsConnected() {
		return eventx.ErrorRegistry.New(eventx.ErrBusNotConnected)
	}
	return nil
}

// GetMetrics returns bus metrics (implements MetricsEventBus)
func (mb *MemoryBus) GetMetrics() eventx.BusMetrics {
	mb.mutex.RLock()
	defer mb.mutex.RUnlock()
	return mb.metrics
}

// PublishAsync publishes an event asynchronously (implements AsyncEventBus)
func (mb *MemoryBus) PublishAsync(ctx context.Context, event eventx.Event) error {
	go func() {
		if err := mb.Publish(ctx, event); err != nil && mb.config.EnableLogging {
			logx.Error("Async publish error for event %s: %v", event.ID(), err)
		}
	}()
	return nil
}

// PublishBatchAsync publishes multiple events asynchronously (implements AsyncEventBus)
func (mb *MemoryBus) PublishBatchAsync(ctx context.Context, events []eventx.Event) error {
	go func() {
		if err := mb.PublishBatch(ctx, events); err != nil && mb.config.EnableLogging {
			logx.Error("Async batch publish error: %v", err)
		}
	}()
	return nil
}
