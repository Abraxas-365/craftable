package eventx

import (
	"encoding/json"
	"time"
)

// SerializableEvent represents an event in a serializable format
type SerializableEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
	Version   string          `json:"version"`
	Data      json.RawMessage `json:"data"`
	Metadata  map[string]any  `json:"metadata"`
}

// ToSerializable converts an event to a serializable format
func ToSerializable(event Event) (*SerializableEvent, error) {
	dataBytes, err := json.Marshal(event.Payload())
	if err != nil {
		return nil, ErrorRegistry.New(ErrSerializationFailed).
			WithCause(err).
			WithDetail("event_id", event.ID()).
			WithDetail("event_type", event.Type())
	}

	return &SerializableEvent{
		ID:        event.ID(),
		Type:      event.Type(),
		Timestamp: event.Timestamp(),
		Source:    event.Source(),
		Version:   event.Version(),
		Data:      json.RawMessage(dataBytes),
		Metadata:  event.Metadata(),
	}, nil
}

// FromSerializable creates a typed event from serializable data
func FromSerializable[T any](se *SerializableEvent) (TypedEvent[T], error) {
	var data T
	if err := json.Unmarshal(se.Data, &data); err != nil {
		return nil, ErrorRegistry.New(ErrSerializationFailed).
			WithCause(err).
			WithDetail("event_id", se.ID).
			WithDetail("event_type", se.Type)
	}

	opts := EventOptions{
		Source:   se.Source,
		Version:  se.Version,
		Metadata: se.Metadata,
	}

	return NewEventWithID(se.ID, se.Type, data, se.Timestamp, opts), nil
}

// ToJSON serializes an event to JSON
func ToJSON(event Event) ([]byte, error) {
	serializable, err := ToSerializable(event)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(serializable)
	if err != nil {
		return nil, ErrorRegistry.New(ErrSerializationFailed).
			WithCause(err).
			WithDetail("event_id", event.ID()).
			WithDetail("event_type", event.Type())
	}

	return data, nil
}

// FromJSON deserializes an event from JSON
func FromJSON[T any](data []byte) (TypedEvent[T], error) {
	var se SerializableEvent
	if err := json.Unmarshal(data, &se); err != nil {
		return nil, ErrorRegistry.New(ErrSerializationFailed).
			WithCause(err).
			WithDetail("operation", "unmarshal_serializable_event")
	}
	return FromSerializable[T](&se)
}
