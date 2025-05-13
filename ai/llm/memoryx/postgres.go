package memoryx

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Abraxas-365/craftable/ai/llm"
	_ "github.com/lib/pq"
)

// PostgresMemory implements the Memory interface with PostgreSQL storage
type PostgresMemory struct {
	db             *sql.DB
	conversationID string
	systemPrompt   string
	maxMessages    int
}

// PostgresMemoryOption configures the PostgreSQL memory
type PostgresMemoryOption func(*PostgresMemory)

// WithPostgresMaxMessages sets the maximum number of messages to retain
func WithPostgresMaxMessages(max int) PostgresMemoryOption {
	return func(m *PostgresMemory) {
		m.maxMessages = max
	}
}

// WithPostgresSystemPrompt sets the system prompt
func WithPostgresSystemPrompt(prompt string) PostgresMemoryOption {
	return func(m *PostgresMemory) {
		m.systemPrompt = prompt
	}
}

// WithConversationID sets a specific conversation ID
func WithConversationID(id string) PostgresMemoryOption {
	return func(m *PostgresMemory) {
		m.conversationID = id
	}
}

// NewPostgresMemory creates a new PostgreSQL-backed memory instance
func NewPostgresMemory(connStr string, opts ...PostgresMemoryOption) (*PostgresMemory, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create a unique conversation ID if not provided
	conversationID := fmt.Sprintf("conv_%d", time.Now().UnixNano())

	m := &PostgresMemory{
		db:             db,
		conversationID: conversationID,
		systemPrompt:   "",
		maxMessages:    100,
	}

	for _, opt := range opts {
		opt(m)
	}

	// Initialize tables
	if err := m.initTables(); err != nil {
		return nil, err
	}

	// Insert system prompt if provided
	if m.systemPrompt != "" {
		ctx := context.Background()
		err := m.storeSystemPrompt(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to store system prompt: %w", err)
		}
	}

	return m, nil
}

// initTables creates necessary tables if they don't exist
func (m *PostgresMemory) initTables() error {
	ctx := context.Background()

	// Create conversations table
	_, err := m.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS conversations (
			id VARCHAR(255) PRIMARY KEY,
			system_prompt TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create conversations table: %w", err)
	}

	// Create messages table
	_, err = m.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS messages (
			id SERIAL PRIMARY KEY,
			conversation_id VARCHAR(255) REFERENCES conversations(id),
			role VARCHAR(50) NOT NULL,
			content TEXT,
			tool_calls JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			message_order INT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	// Insert conversation record if it doesn't exist
	_, err = m.db.ExecContext(ctx, `
		INSERT INTO conversations (id, system_prompt)
		VALUES ($1, $2)
		ON CONFLICT (id) DO NOTHING
	`, m.conversationID, m.systemPrompt)
	if err != nil {
		return fmt.Errorf("failed to initialize conversation: %w", err)
	}

	return nil
}

// storeSystemPrompt stores or updates the system prompt for the current conversation
func (m *PostgresMemory) storeSystemPrompt(ctx context.Context) error {
	// Update the conversations table
	_, err := m.db.ExecContext(ctx, `
		UPDATE conversations 
		SET system_prompt = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, m.systemPrompt, m.conversationID)
	if err != nil {
		return fmt.Errorf("failed to update system prompt: %w", err)
	}

	// Check if system message already exists
	var count int
	err = m.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM messages 
		WHERE conversation_id = $1 AND role = $2
	`, m.conversationID, llm.RoleSystem).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for system message: %w", err)
	}

	// If system message exists, update it
	if count > 0 {
		_, err = m.db.ExecContext(ctx, `
			UPDATE messages 
			SET content = $1, updated_at = CURRENT_TIMESTAMP
			WHERE conversation_id = $2 AND role = $3
		`, m.systemPrompt, m.conversationID, llm.RoleSystem)
		if err != nil {
			return fmt.Errorf("failed to update system message: %w", err)
		}
	} else if m.systemPrompt != "" {
		// If no system message but we have a system prompt, insert it
		_, err = m.db.ExecContext(ctx, `
			INSERT INTO messages (conversation_id, role, content, message_order)
			VALUES ($1, $2, $3, 0)
		`, m.conversationID, llm.RoleSystem, m.systemPrompt)
		if err != nil {
			return fmt.Errorf("failed to insert system message: %w", err)
		}
	}

	return nil
}

// Messages returns all messages including system prompt
func (m *PostgresMemory) Messages() ([]llm.Message, error) {
	ctx := context.Background()

	// Query for messages
	rows, err := m.db.QueryContext(ctx, `
		SELECT role, content, tool_calls 
		FROM messages 
		WHERE conversation_id = $1
		ORDER BY message_order
	`, m.conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []llm.Message
	for rows.Next() {
		var role, content string
		var toolCallsJSON sql.NullString

		if err := rows.Scan(&role, &content, &toolCallsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}

		message := llm.Message{
			Role:    role,
			Content: content,
		}

		// Parse tool calls if present
		if toolCallsJSON.Valid && toolCallsJSON.String != "" {
			var toolCalls []llm.ToolCall
			if err := json.Unmarshal([]byte(toolCallsJSON.String), &toolCalls); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tool calls: %w", err)
			}
			message.ToolCalls = toolCalls
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message rows: %w", err)
	}

	return messages, nil
}

// Add adds a new message to memory
func (m *PostgresMemory) Add(message llm.Message) error {
	ctx := context.Background()

	// Get next message order
	var nextOrder int
	err := m.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(message_order), -1) + 1 
		FROM messages 
		WHERE conversation_id = $1
	`, m.conversationID).Scan(&nextOrder)
	if err != nil {
		return fmt.Errorf("failed to get next message order: %w", err)
	}

	// Marshal tool calls to JSON if present
	var toolCallsJSON sql.NullString
	if len(message.ToolCalls) > 0 {
		toolCallsBytes, err := json.Marshal(message.ToolCalls)
		if err != nil {
			return fmt.Errorf("failed to marshal tool calls: %w", err)
		}
		toolCallsJSON = sql.NullString{
			String: string(toolCallsBytes),
			Valid:  true,
		}
	}

	// Insert message
	_, err = m.db.ExecContext(ctx, `
		INSERT INTO messages (conversation_id, role, content, tool_calls, message_order)
		VALUES ($1, $2, $3, $4, $5)
	`, m.conversationID, message.Role, message.Content, toolCallsJSON, nextOrder)
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	// Update conversation timestamp
	_, err = m.db.ExecContext(ctx, `
		UPDATE conversations 
		SET updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, m.conversationID)
	if err != nil {
		return fmt.Errorf("failed to update conversation timestamp: %w", err)
	}

	// If we've exceeded max messages, trim the oldest non-system messages
	if m.maxMessages > 0 {
		_, err = m.db.ExecContext(ctx, `
			DELETE FROM messages 
			WHERE id IN (
				SELECT id FROM messages
				WHERE conversation_id = $1 AND role != $2
				ORDER BY message_order
				LIMIT (SELECT COUNT(*) FROM messages WHERE conversation_id = $1) - $3
			)
		`, m.conversationID, llm.RoleSystem, m.maxMessages)
		if err != nil {
			return fmt.Errorf("failed to trim old messages: %w", err)
		}
	}

	return nil
}

// Clear resets the conversation but keeps the system prompt
func (m *PostgresMemory) Clear() error {
	ctx := context.Background()

	// Delete all non-system messages
	_, err := m.db.ExecContext(ctx, `
		DELETE FROM messages 
		WHERE conversation_id = $1 AND role != $2
	`, m.conversationID, llm.RoleSystem)
	if err != nil {
		return fmt.Errorf("failed to clear messages: %w", err)
	}

	return nil
}

// SystemPrompt gets the current system prompt
func (m *PostgresMemory) SystemPrompt() (string, error) {
	ctx := context.Background()

	var prompt string
	err := m.db.QueryRowContext(ctx, `
		SELECT system_prompt FROM conversations WHERE id = $1
	`, m.conversationID).Scan(&prompt)
	if err != nil {
		return "", fmt.Errorf("failed to get system prompt: %w", err)
	}

	m.systemPrompt = prompt
	return prompt, nil
}

// UpdateSystemPrompt updates the system prompt
func (m *PostgresMemory) UpdateSystemPrompt(content string) error {
	ctx := context.Background()

	m.systemPrompt = content
	return m.storeSystemPrompt(ctx)
}

// Close closes the database connection
func (m *PostgresMemory) Close() error {
	return m.db.Close()
}

// ListConversations returns all conversation IDs in the database
func (m *PostgresMemory) ListConversations() ([]string, error) {
	ctx := context.Background()

	rows, err := m.db.QueryContext(ctx, `
		SELECT id FROM conversations ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	var conversations []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan conversation ID: %w", err)
		}
		conversations = append(conversations, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating conversation rows: %w", err)
	}

	return conversations, nil
}

// SwitchConversation changes to a different conversation
func (m *PostgresMemory) SwitchConversation(conversationID string) error {
	ctx := context.Background()

	// Check if conversation exists
	var exists bool
	err := m.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM conversations WHERE id = $1)
	`, conversationID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if conversation exists: %w", err)
	}

	// Create conversation if it doesn't exist
	if !exists {
		_, err = m.db.ExecContext(ctx, `
			INSERT INTO conversations (id, system_prompt)
			VALUES ($1, $2)
		`, conversationID, m.systemPrompt)
		if err != nil {
			return fmt.Errorf("failed to create new conversation: %w", err)
		}
	} else {
		// Update local system prompt from database
		err = m.db.QueryRowContext(ctx, `
			SELECT system_prompt FROM conversations WHERE id = $1
		`, conversationID).Scan(&m.systemPrompt)
		if err != nil {
			return fmt.Errorf("failed to get system prompt for conversation: %w", err)
		}
	}

	m.conversationID = conversationID
	return nil
}
