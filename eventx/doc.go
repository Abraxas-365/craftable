// Package eventx provides a flexible event system with pluggable backends.
//
// The package defines core interfaces for events and event buses, allowing
// different implementations like in-memory, RabbitMQ, AWS SQS, etc.
//
// Basic usage:
//
//	// Create an event
//	userEvent := eventx.NewEvent("user.created", UserData{
//		ID: "user-123",
//		Name: "John Doe",
//	})
//
//	// Create event bus (in-memory example)
//	bus := eventxmemory.New()
//
//	// Subscribe to events
//	eventx.SubscribeTyped(bus, "user.created", handleUserCreated)
//
//	// Publish events
//	ctx := context.Background()
//	if err := bus.Publish(ctx, userEvent); err != nil {
//		log.Printf("Error: %v", err)
//	}
package eventx
