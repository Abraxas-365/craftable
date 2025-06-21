package eventxsqs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Abraxas-365/craftable/eventx"
	"github.com/Abraxas-365/craftable/logx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// SQSBus implements EventBus interface using AWS SQS
type SQSBus struct {
	config    SQSConfig
	client    *sqs.Client
	handlers  map[string][]eventx.EventHandler
	filters   map[string][]eventx.EventFilter
	metrics   eventx.BusMetrics
	mutex     sync.RWMutex
	connected bool
	queues    map[string]*QueueInfo
	consumers map[string]context.CancelFunc
	awsConfig aws.Config
}

// QueueInfo stores information about SQS queues
type QueueInfo struct {
	URL        string
	Name       string
	Attributes map[string]string
}

// SQSConfig extends BusConfig with SQS-specific settings
type SQSConfig struct {
	eventx.BusConfig

	// AWS Configuration
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	SessionToken    string `json:"session_token"`
	Profile         string `json:"profile"`

	// SQS specific
	QueuePrefix           string `json:"queue_prefix"`
	MessageRetentionDays  int    `json:"message_retention_days"`
	VisibilityTimeoutSecs int    `json:"visibility_timeout_seconds"`
	ReceiveTimeoutSecs    int    `json:"receive_timeout_seconds"`
	MaxReceiveCount       int    `json:"max_receive_count"`
	EnableLongPolling     bool   `json:"enable_long_polling"`
	WaitTimeSeconds       int    `json:"wait_time_seconds"`
	EnableFIFO            bool   `json:"enable_fifo"`
	ContentBasedDedup     bool   `json:"content_based_deduplication"`

	// Dead Letter Queue
	EnableDLQ bool   `json:"enable_dlq"`
	DLQSuffix string `json:"dlq_suffix"`

	// Batch settings
	MaxBatchSize  int `json:"max_batch_size"`
	BatchWaitTime int `json:"batch_wait_time_ms"`

	// Consumer settings
	MaxConcurrentConsumers  int `json:"max_concurrent_consumers"`
	ConsumerPollingInterval int `json:"consumer_polling_interval_ms"`
}

// DefaultSQSConfig returns default SQS configuration
func DefaultSQSConfig() SQSConfig {
	return SQSConfig{
		BusConfig:               eventx.DefaultBusConfig(),
		Region:                  "us-east-1",
		QueuePrefix:             "eventx",
		MessageRetentionDays:    14,
		VisibilityTimeoutSecs:   30,
		ReceiveTimeoutSecs:      20,
		MaxReceiveCount:         3,
		EnableLongPolling:       true,
		WaitTimeSeconds:         20,
		EnableFIFO:              false,
		ContentBasedDedup:       false,
		EnableDLQ:               true,
		DLQSuffix:               "dlq",
		MaxBatchSize:            10,
		BatchWaitTime:           100,
		MaxConcurrentConsumers:  5,
		ConsumerPollingInterval: 1000,
	}
}

// New creates a new SQS event bus
func New(config SQSConfig) eventx.EventBus {
	return &SQSBus{
		config:    config,
		handlers:  make(map[string][]eventx.EventHandler),
		filters:   make(map[string][]eventx.EventFilter),
		metrics:   eventx.BusMetrics{},
		queues:    make(map[string]*QueueInfo),
		consumers: make(map[string]context.CancelFunc),
	}
}

// Connect establishes connection to AWS SQS
func (sb *SQSBus) Connect(ctx context.Context) error {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()

	if sb.connected {
		return nil
	}

	// Load AWS configuration
	var err error
	if sb.config.Profile != "" {
		sb.awsConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(sb.config.Region),
			config.WithSharedConfigProfile(sb.config.Profile),
		)
	} else if sb.config.AccessKeyID != "" && sb.config.SecretAccessKey != "" {
		sb.awsConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(sb.config.Region),
			config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     sb.config.AccessKeyID,
					SecretAccessKey: sb.config.SecretAccessKey,
					SessionToken:    sb.config.SessionToken,
				}, nil
			})),
		)
	} else {
		sb.awsConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(sb.config.Region),
		)
	}

	if err != nil {
		return eventx.ErrorRegistry.New(eventx.ErrConnectionFailed).
			WithCause(err).
			WithDetail("region", sb.config.Region)
	}

	// Create SQS client
	sb.client = sqs.NewFromConfig(sb.awsConfig)

	// Test connection by listing queues
	_, err = sb.client.ListQueues(ctx, &sqs.ListQueuesInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		return eventx.ErrorRegistry.New(eventx.ErrConnectionFailed).
			WithCause(err).
			WithDetail("operation", "test_connection")
	}

	sb.connected = true
	sb.metrics.ConnectionStatus = true

	if sb.config.EnableLogging {
		logx.Debug("Connecting to AWS SQS in region: %s", sb.config.Region)
	}

	return nil
}

// Disconnect closes connection to AWS SQS
func (sb *SQSBus) Disconnect(ctx context.Context) error {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()

	if !sb.connected {
		return nil
	}

	// Cancel all consumers
	for eventType, cancel := range sb.consumers {
		cancel()
		if sb.config.EnableLogging {
			logx.Debug("Stopping consumer for event type: %s", eventType)
		}
	}
	sb.consumers = make(map[string]context.CancelFunc)

	sb.client = nil
	sb.connected = false
	sb.metrics.ConnectionStatus = false

	if sb.config.EnableLogging {
		logx.Debug("Disconnecting from AWS SQS")
	}

	return nil
}

// IsConnected returns connection status
func (sb *SQSBus) IsConnected() bool {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()
	return sb.connected
}

// Subscribe registers an event handler
func (sb *SQSBus) Subscribe(ctx context.Context, eventType string, handler eventx.EventHandler) error {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()

	if !sb.connected {
		return eventx.ErrorRegistry.New(eventx.ErrBusNotConnected)
	}

	// Create queue for this event type if it doesn't exist
	queueInfo, err := sb.ensureQueue(ctx, eventType)
	if err != nil {
		return err
	}

	// Store handler
	sb.handlers[eventType] = append(sb.handlers[eventType], handler)
	sb.queues[eventType] = queueInfo
	sb.metrics.ActiveSubscribers++

	// Start consumer if not already running
	if _, exists := sb.consumers[eventType]; !exists {
		consumerCtx, cancel := context.WithCancel(context.Background())
		sb.consumers[eventType] = cancel

		// Start multiple consumer goroutines for concurrent processing
		for range sb.config.MaxConcurrentConsumers {
			go sb.consumeMessages(consumerCtx, eventType, queueInfo)
		}
	}

	if sb.config.EnableLogging {
		logx.Debug("Subscribing to event type: %s (queue: %s)", eventType, queueInfo.Name)
	}

	return nil
}

// ensureQueue creates a queue if it doesn't exist
func (sb *SQSBus) ensureQueue(ctx context.Context, eventType string) (*QueueInfo, error) {
	queueName := sb.getQueueName(eventType)

	// Check if queue already exists
	getQueueOutput, err := sb.client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})

	if err == nil {
		// Queue exists, get its attributes
		attrs, err := sb.client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
			QueueUrl:       getQueueOutput.QueueUrl,
			AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameAll},
		})
		if err != nil {
			return nil, eventx.ErrorRegistry.New(eventx.ErrSubscriptionFailed).
				WithCause(err).
				WithDetail("operation", "get_queue_attributes").
				WithDetail("queue_name", queueName)
		}

		return &QueueInfo{
			URL:        *getQueueOutput.QueueUrl,
			Name:       queueName,
			Attributes: attrs.Attributes,
		}, nil
	}

	// Queue doesn't exist, create it
	queueAttributes := sb.buildQueueAttributes(eventType)

	createOutput, err := sb.client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName:  aws.String(queueName),
		Attributes: queueAttributes,
	})
	if err != nil {
		return nil, eventx.ErrorRegistry.New(eventx.ErrSubscriptionFailed).
			WithCause(err).
			WithDetail("operation", "create_queue").
			WithDetail("queue_name", queueName)
	}

	// Create DLQ if enabled
	if sb.config.EnableDLQ {
		dlqName := queueName + "-" + sb.config.DLQSuffix
		dlqOutput, err := sb.client.CreateQueue(ctx, &sqs.CreateQueueInput{
			QueueName: aws.String(dlqName),
			Attributes: map[string]string{
				string(types.QueueAttributeNameMessageRetentionPeriod): strconv.Itoa(sb.config.MessageRetentionDays * 24 * 3600),
			},
		})
		if err != nil {
			logx.Error("Failed to create DLQ %s: %v", dlqName, err)
		} else if sb.config.EnableLogging {
			logx.Debug("Created DLQ: %s", *dlqOutput.QueueUrl)
		}
	}

	if sb.config.EnableLogging {
		logx.Debug("Creating SQS queue: %s with attributes: %v", queueName, queueAttributes)
	}

	return &QueueInfo{
		URL:        *createOutput.QueueUrl,
		Name:       queueName,
		Attributes: queueAttributes,
	}, nil
}

// buildQueueAttributes builds SQS queue attributes based on configuration
func (sb *SQSBus) buildQueueAttributes(eventType string) map[string]string {
	attrs := map[string]string{
		string(types.QueueAttributeNameMessageRetentionPeriod):        strconv.Itoa(sb.config.MessageRetentionDays * 24 * 3600),
		string(types.QueueAttributeNameVisibilityTimeout):             strconv.Itoa(sb.config.VisibilityTimeoutSecs),
		string(types.QueueAttributeNameReceiveMessageWaitTimeSeconds): strconv.Itoa(sb.config.WaitTimeSeconds),
	}

	if sb.config.EnableDLQ {
		dlqName := sb.getQueueName(eventType) + "-" + sb.config.DLQSuffix
		// Note: In real implementation, you'd need to get the DLQ ARN
		redrivePolicy := fmt.Sprintf(`{"deadLetterTargetArn":"arn:aws:sqs:%s:*:%s","maxReceiveCount":%d}`,
			sb.config.Region, dlqName, sb.config.MaxReceiveCount)
		attrs[string(types.QueueAttributeNameRedrivePolicy)] = redrivePolicy
	}

	if sb.config.EnableFIFO {
		attrs[string(types.QueueAttributeNameFifoQueue)] = "true"
		if sb.config.ContentBasedDedup {
			attrs[string(types.QueueAttributeNameContentBasedDeduplication)] = "true"
		}
	}

	return attrs
}

// getQueueName generates the queue name for an event type
func (sb *SQSBus) getQueueName(eventType string) string {
	queueName := fmt.Sprintf("%s-%s", sb.config.QueuePrefix, eventType)
	if sb.config.EnableFIFO {
		queueName += ".fifo"
	}
	return queueName
}

// consumeMessages consumes messages from SQS queue
func (sb *SQSBus) consumeMessages(ctx context.Context, eventType string, queueInfo *QueueInfo) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			sb.processQueueMessages(ctx, eventType, queueInfo)
		}

		// Wait before next polling cycle
		time.Sleep(time.Duration(sb.config.ConsumerPollingInterval) * time.Millisecond)
	}
}

// processQueueMessages processes a batch of messages from the queue
func (sb *SQSBus) processQueueMessages(ctx context.Context, eventType string, queueInfo *QueueInfo) {
	// Receive messages
	receiveOutput, err := sb.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(queueInfo.URL),
		MaxNumberOfMessages:   int32(sb.config.MaxBatchSize),
		WaitTimeSeconds:       int32(sb.config.ReceiveTimeoutSecs),
		MessageAttributeNames: []string{"All"},
		AttributeNames:        []types.QueueAttributeName{types.QueueAttributeNameAll},
	})
	if err != nil {
		if sb.config.EnableLogging {
			logx.Error("Error receiving messages from queue %s: %v", queueInfo.Name, err)
		}
		return
	}

	if len(receiveOutput.Messages) == 0 {
		return
	}

	// Process each message
	var messagesToDelete []types.DeleteMessageBatchRequestEntry

	for _, msg := range receiveOutput.Messages {
		if sb.processMessage(ctx, eventType, msg) {
			// Mark for deletion if processing succeeded
			messagesToDelete = append(messagesToDelete, types.DeleteMessageBatchRequestEntry{
				Id:            msg.MessageId,
				ReceiptHandle: msg.ReceiptHandle,
			})
		}
	}

	// Delete successfully processed messages
	if len(messagesToDelete) > 0 {
		_, err := sb.client.DeleteMessageBatch(ctx, &sqs.DeleteMessageBatchInput{
			QueueUrl: aws.String(queueInfo.URL),
			Entries:  messagesToDelete,
		})
		if err != nil && sb.config.EnableLogging {
			logx.Error("Error deleting messages from queue %s: %v", queueInfo.Name, err)
		}
	}
}

// processMessage processes a single SQS message
func (sb *SQSBus) processMessage(_ context.Context, eventType string, msg types.Message) bool {
	// Deserialize event
	var serializableEvent eventx.SerializableEvent
	if err := json.Unmarshal([]byte(*msg.Body), &serializableEvent); err != nil {
		if sb.config.EnableLogging {
			logx.Error("Failed to deserialize message body: %s, error: %v", *msg.Body, err)
		}
		return false // Don't delete malformed messages
	}

	// Create generic event
	event := &eventx.BaseEvent[json.RawMessage]{}
	*event = eventx.BaseEvent[json.RawMessage]{
		// Initialize fields from serializableEvent
	}

	// Apply filters
	sb.mutex.RLock()
	filters := make([]eventx.EventFilter, len(sb.filters[eventType]))
	copy(filters, sb.filters[eventType])
	sb.mutex.RUnlock()

	for _, filter := range filters {
		if !filter(event) {
			return true // Skip but delete the message
		}
	}

	// Execute handlers
	sb.mutex.RLock()
	handlers := make([]eventx.EventHandler, len(sb.handlers[eventType]))
	copy(handlers, sb.handlers[eventType])
	sb.mutex.RUnlock()

	success := true
	for _, handler := range handlers {
		if err := handler(event); err != nil {
			sb.mutex.Lock()
			sb.metrics.EventsFailed++
			sb.mutex.Unlock()

			if sb.config.EnableLogging {
				logx.Error("Error handling event %s: %v", serializableEvent.ID, err)
			}
			success = false
		} else {
			sb.mutex.Lock()
			sb.metrics.EventsProcessed++
			sb.mutex.Unlock()
		}
	}

	return success
}

// Publish publishes an event
func (sb *SQSBus) Publish(ctx context.Context, event eventx.Event) error {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	if !sb.connected {
		return eventx.ErrorRegistry.New(eventx.ErrBusNotConnected)
	}

	// Ensure queue exists
	queueInfo, err := sb.ensureQueue(ctx, event.Type())
	if err != nil {
		return err
	}

	// Serialize event
	data, err := eventx.ToJSON(event)
	if err != nil {
		return err
	}

	// Prepare message attributes
	messageAttributes := make(map[string]types.MessageAttributeValue)
	messageAttributes["EventType"] = types.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String(event.Type()),
	}
	messageAttributes["EventSource"] = types.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String(event.Source()),
	}
	messageAttributes["EventVersion"] = types.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String(event.Version()),
	}

	// Add custom metadata as message attributes
	for key, value := range event.Metadata() {
		if strValue, ok := value.(string); ok {
			messageAttributes[key] = types.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(strValue),
			}
		}
	}

	// Prepare send message input
	sendInput := &sqs.SendMessageInput{
		QueueUrl:          aws.String(queueInfo.URL),
		MessageBody:       aws.String(string(data)),
		MessageAttributes: messageAttributes,
	}

	// Add FIFO-specific attributes if enabled
	if sb.config.EnableFIFO {
		sendInput.MessageGroupId = aws.String(event.Type())
		if !sb.config.ContentBasedDedup {
			sendInput.MessageDeduplicationId = aws.String(event.ID())
		}
	}

	// Send message
	_, err = sb.client.SendMessage(ctx, sendInput)
	if err != nil {
		sb.mutex.Lock()
		sb.metrics.EventsFailed++
		sb.mutex.Unlock()

		return eventx.ErrorRegistry.New(eventx.ErrPublishFailed).
			WithCause(err).
			WithDetail("event_id", event.ID()).
			WithDetail("event_type", event.Type()).
			WithDetail("queue_url", queueInfo.URL)
	}

	sb.mutex.Lock()
	sb.metrics.EventsPublished++
	sb.mutex.Unlock()

	return nil
}

// PublishBatch publishes multiple events in a batch
func (sb *SQSBus) PublishBatch(ctx context.Context, events []eventx.Event) error {
	if len(events) == 0 {
		return nil
	}

	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	if !sb.connected {
		return eventx.ErrorRegistry.New(eventx.ErrBusNotConnected)
	}

	// Group events by type for batch sending
	eventsByType := make(map[string][]eventx.Event)
	for _, event := range events {
		eventsByType[event.Type()] = append(eventsByType[event.Type()], event)
	}

	var lastErr error
	for eventType, typeEvents := range eventsByType {
		// Process events in batches of MaxBatchSize
		for i := 0; i < len(typeEvents); i += sb.config.MaxBatchSize {
			end := min(i+sb.config.MaxBatchSize, len(typeEvents))

			batch := typeEvents[i:end]
			if err := sb.sendBatch(ctx, eventType, batch); err != nil {
				lastErr = err
			}
		}
	}

	return lastErr
}

// sendBatch sends a batch of events of the same type
func (sb *SQSBus) sendBatch(ctx context.Context, eventType string, events []eventx.Event) error {
	// Ensure queue exists
	queueInfo, err := sb.ensureQueue(ctx, eventType)
	if err != nil {
		return err
	}

	// Prepare batch entries
	var entries []types.SendMessageBatchRequestEntry
	for i, event := range events {
		// Serialize event
		data, err := eventx.ToJSON(event)
		if err != nil {
			continue // Skip invalid events
		}

		// Prepare message attributes
		messageAttributes := make(map[string]types.MessageAttributeValue)
		messageAttributes["EventType"] = types.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(event.Type()),
		}
		messageAttributes["EventSource"] = types.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(event.Source()),
		}

		entry := types.SendMessageBatchRequestEntry{
			Id:                aws.String(fmt.Sprintf("msg-%d", i)),
			MessageBody:       aws.String(string(data)),
			MessageAttributes: messageAttributes,
		}

		// Add FIFO-specific attributes if enabled
		if sb.config.EnableFIFO {
			entry.MessageGroupId = aws.String(event.Type())
			if !sb.config.ContentBasedDedup {
				entry.MessageDeduplicationId = aws.String(event.ID())
			}
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil
	}

	// Send batch
	output, err := sb.client.SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
		QueueUrl: aws.String(queueInfo.URL),
		Entries:  entries,
	})
	if err != nil {
		sb.mutex.Lock()
		sb.metrics.EventsFailed += int64(len(entries))
		sb.mutex.Unlock()

		return eventx.ErrorRegistry.New(eventx.ErrPublishFailed).
			WithCause(err).
			WithDetail("event_type", eventType).
			WithDetail("batch_size", len(entries)).
			WithDetail("queue_url", queueInfo.URL)
	}

	// Update metrics
	sb.mutex.Lock()
	sb.metrics.EventsPublished += int64(len(output.Successful))
	sb.metrics.EventsFailed += int64(len(output.Failed))
	sb.mutex.Unlock()

	// Log failures
	if len(output.Failed) > 0 && sb.config.EnableLogging {
		for _, failed := range output.Failed {
			logx.Error("Failed to send message %s: %s", *failed.Id, *failed.Message)
		}
	}

	return nil
}

// Unsubscribe removes handlers for an event type
func (sb *SQSBus) Unsubscribe(ctx context.Context, eventType string) error {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()

	// Cancel consumer
	if cancel, exists := sb.consumers[eventType]; exists {
		cancel()
		delete(sb.consumers, eventType)
	}

	// Remove handlers and queues
	if handlers, exists := sb.handlers[eventType]; exists {
		sb.metrics.ActiveSubscribers -= len(handlers)
		delete(sb.handlers, eventType)
		delete(sb.filters, eventType)
		delete(sb.queues, eventType)
	}

	if sb.config.EnableLogging {
		logx.Debug("Unsubscribing from event type: %s", eventType)
	}

	return nil
}

// AddFilter adds a filter for an event type
func (sb *SQSBus) AddFilter(eventType string, filter eventx.EventFilter) error {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()
	sb.filters[eventType] = append(sb.filters[eventType], filter)
	return nil
}

// RemoveFilter removes filters for an event type
func (sb *SQSBus) RemoveFilter(eventType string) error {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()
	delete(sb.filters, eventType)
	return nil
}

// ListEventTypes returns all registered event types
func (sb *SQSBus) ListEventTypes() []string {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	types := make([]string, 0, len(sb.handlers))
	for eventType := range sb.handlers {
		types = append(types, eventType)
	}
	return types
}

// HandlerCount returns the number of handlers for an event type
func (sb *SQSBus) HandlerCount(eventType string) int {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()
	return len(sb.handlers[eventType])
}

// Health returns the health status
func (sb *SQSBus) Health(ctx context.Context) error {
	if !sb.IsConnected() {
		return eventx.ErrorRegistry.New(eventx.ErrBusNotConnected)
	}

	// Test SQS connectivity
	_, err := sb.client.ListQueues(ctx, &sqs.ListQueuesInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		return eventx.ErrorRegistry.New(eventx.ErrConnectionFailed).
			WithCause(err).
			WithDetail("operation", "health_check")
	}

	return nil
}

// GetMetrics returns bus metrics
func (sb *SQSBus) GetMetrics() eventx.BusMetrics {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()
	return sb.metrics
}

// PublishAsync publishes an event asynchronously
func (sb *SQSBus) PublishAsync(ctx context.Context, event eventx.Event) error {
	go func() {
		if err := sb.Publish(ctx, event); err != nil && sb.config.EnableLogging {
			logx.Error("Async publish error for event %s: %v", event.ID(), err)
		}
	}()
	return nil
}

// PublishBatchAsync publishes multiple events asynchronously
func (sb *SQSBus) PublishBatchAsync(ctx context.Context, events []eventx.Event) error {
	go func() {
		if err := sb.PublishBatch(ctx, events); err != nil && sb.config.EnableLogging {
			logx.Error("Async batch publish error: %v", err)
		}
	}()
	return nil
}

// GetQueueInfo returns information about a queue for an event type
func (sb *SQSBus) GetQueueInfo(eventType string) (*QueueInfo, bool) {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	info, exists := sb.queues[eventType]
	return info, exists
}

// PurgeQueue purges all messages from a queue (useful for testing)
func (sb *SQSBus) PurgeQueue(ctx context.Context, eventType string) error {
	sb.mutex.RLock()
	queueInfo, exists := sb.queues[eventType]
	sb.mutex.RUnlock()

	if !exists {
		return eventx.ErrorRegistry.New(eventx.ErrEventNotFound).
			WithDetail("event_type", eventType)
	}

	_, err := sb.client.PurgeQueue(ctx, &sqs.PurgeQueueInput{
		QueueUrl: aws.String(queueInfo.URL),
	})
	if err != nil {
		return eventx.ErrorRegistry.New(eventx.ErrConnectionFailed).
			WithCause(err).
			WithDetail("operation", "purge_queue").
			WithDetail("queue_url", queueInfo.URL)
	}

	return nil
}
