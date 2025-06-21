package eventx

import (
	"context"
	"sync"
)

// BusRegistry manages multiple event bus instances
type BusRegistry struct {
	buses map[string]EventBus
	mutex sync.RWMutex
}

// NewBusRegistry creates a new bus registry
func NewBusRegistry() *BusRegistry {
	return &BusRegistry{
		buses: make(map[string]EventBus),
	}
}

// Register adds a new event bus to the registry
func (r *BusRegistry) Register(name string, bus EventBus) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.buses[name]; exists {
		return ErrorRegistry.New(ErrInvalidConfiguration).
			WithDetail("bus_name", name).
			WithDetail("reason", "bus already registered")
	}

	r.buses[name] = bus
	return nil
}

// Get retrieves an event bus by name
func (r *BusRegistry) Get(name string) (EventBus, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	bus, exists := r.buses[name]
	if !exists {
		return nil, ErrorRegistry.New(ErrEventNotFound).
			WithDetail("bus_name", name)
	}

	return bus, nil
}

// Remove removes an event bus from the registry
func (r *BusRegistry) Remove(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.buses[name]; !exists {
		return ErrorRegistry.New(ErrEventNotFound).
			WithDetail("bus_name", name)
	}

	delete(r.buses, name)
	return nil
}

// List returns all registered bus names
func (r *BusRegistry) List() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.buses))
	for name := range r.buses {
		names = append(names, name)
	}
	return names
}

// ConnectAll connects all registered buses
func (r *BusRegistry) ConnectAll(ctx context.Context) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for name, bus := range r.buses {
		if err := bus.Connect(ctx); err != nil {
			return ErrorRegistry.New(ErrConnectionFailed).
				WithCause(err).
				WithDetail("bus_name", name)
		}
	}
	return nil
}

// DisconnectAll disconnects all registered buses
func (r *BusRegistry) DisconnectAll(ctx context.Context) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var lastErr error
	for name, bus := range r.buses {
		if err := bus.Disconnect(ctx); err != nil {
			lastErr = ErrorRegistry.New(ErrConnectionFailed).
				WithCause(err).
				WithDetail("bus_name", name)
		}
	}
	return lastErr
}

// Global registry instance
var GlobalRegistry = NewBusRegistry()
