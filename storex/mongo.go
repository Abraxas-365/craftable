package storex

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TypedMongo provides MongoDB operations for a specific type
type TypedMongo[T any] struct {
	Collection *mongo.Collection
	IDField    string // Field name for the ID (default: "_id")
}

// NewTypedMongo creates a new TypedMongo helper for a specific type
func NewTypedMongo[T any](collection *mongo.Collection) *TypedMongo[T] {
	return &TypedMongo[T]{
		Collection: collection,
		IDField:    "_id",
	}
}

// WithIDField sets a custom ID field name
func (m *TypedMongo[T]) WithIDField(fieldName string) *TypedMongo[T] {
	m.IDField = fieldName
	return m
}

// Create adds a new document to the collection
func (m *TypedMongo[T]) Create(ctx context.Context, item T) (T, error) {
	result, err := m.Collection.InsertOne(ctx, item)
	if err != nil {
		return item, storeErrors.New(ErrMongoInsertFailed).
			WithDetail("collection", m.Collection.Name()).
			WithCause(err)
	}

	// If the item has an ID field, we need to fetch the complete document
	if result.InsertedID != nil {
		filter := bson.M{m.IDField: result.InsertedID}
		err = m.Collection.FindOne(ctx, filter).Decode(&item)
		if err != nil {
			return item, storeErrors.New(ErrMongoFindFailed).
				WithDetail("id", fmt.Sprintf("%v", result.InsertedID)).
				WithDetail("collection", m.Collection.Name()).
				WithCause(err)
		}
	}

	return item, nil
}

// FindByID retrieves a document by ID
func (m *TypedMongo[T]) FindByID(ctx context.Context, id string) (T, error) {
	var result T
	var objectID interface{} = id

	// If using ObjectIDs, convert string to ObjectID
	if m.IDField == "_id" {
		var err error
		objectID, err = primitive.ObjectIDFromHex(id)
		if err != nil {
			return result, storeErrors.New(ErrInvalidID).
				WithDetail("id", id).
				WithCause(err)
		}
	}

	err := m.Collection.FindOne(ctx, bson.M{m.IDField: objectID}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return result, storeErrors.New(ErrRecordNotFound).
				WithDetail("id", id).
				WithDetail("collection", m.Collection.Name())
		}
		return result, storeErrors.New(ErrMongoFindFailed).
			WithDetail("id", id).
			WithDetail("collection", m.Collection.Name()).
			WithCause(err)
	}

	return result, nil
}

// FindOne retrieves a single document matching the filter
func (m *TypedMongo[T]) FindOne(ctx context.Context, filter map[string]any) (T, error) {
	var result T
	bsonFilter := convertMapToBson(filter)

	err := m.Collection.FindOne(ctx, bsonFilter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return result, storeErrors.New(ErrRecordNotFound).
				WithDetail("filter", fmt.Sprintf("%v", filter)).
				WithDetail("collection", m.Collection.Name())
		}
		return result, storeErrors.New(ErrMongoFindFailed).
			WithDetail("filter", fmt.Sprintf("%v", filter)).
			WithDetail("collection", m.Collection.Name()).
			WithCause(err)
	}

	return result, nil
}

// Update modifies an existing document
func (m *TypedMongo[T]) Update(ctx context.Context, id string, item T) (T, error) {
	var objectID interface{} = id

	// If using ObjectIDs, convert string to ObjectID
	if m.IDField == "_id" {
		var err error
		objectID, err = primitive.ObjectIDFromHex(id)
		if err != nil {
			return item, storeErrors.New(ErrInvalidID).
				WithDetail("id", id).
				WithCause(err)
		}
	}

	// Create update document
	update := bson.M{"$set": item}

	// Execute update
	result, err := m.Collection.UpdateOne(
		ctx,
		bson.M{m.IDField: objectID},
		update,
	)
	if err != nil {
		return item, storeErrors.New(ErrMongoUpdateFailed).
			WithDetail("id", id).
			WithDetail("collection", m.Collection.Name()).
			WithCause(err)
	}

	if result.MatchedCount == 0 {
		return item, storeErrors.New(ErrRecordNotFound).
			WithDetail("id", id).
			WithDetail("collection", m.Collection.Name())
	}

	// Fetch the updated document
	return m.FindByID(ctx, id)
}

// Delete removes a document from the collection
func (m *TypedMongo[T]) Delete(ctx context.Context, id string) error {
	var objectID interface{} = id

	// If using ObjectIDs, convert string to ObjectID
	if m.IDField == "_id" {
		var err error
		objectID, err = primitive.ObjectIDFromHex(id)
		if err != nil {
			return storeErrors.New(ErrInvalidID).
				WithDetail("id", id).
				WithCause(err)
		}
	}

	result, err := m.Collection.DeleteOne(ctx, bson.M{m.IDField: objectID})
	if err != nil {
		return storeErrors.New(ErrMongoDeleteFailed).
			WithDetail("id", id).
			WithDetail("collection", m.Collection.Name()).
			WithCause(err)
	}

	if result.DeletedCount == 0 {
		return storeErrors.New(ErrRecordNotFound).
			WithDetail("id", id).
			WithDetail("collection", m.Collection.Name())
	}

	return nil
}

// Paginate retrieves documents with pagination
func (m *TypedMongo[T]) Paginate(ctx context.Context, opts PaginationOptions) (Paginated[T], error) {
	return PaginateMongo[T](ctx, m.Collection, opts)
}

// BulkInsert adds multiple documents in a single operation
func (m *TypedMongo[T]) BulkInsert(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	// Convert items to interface slice
	docs := make([]interface{}, len(items))
	for i, item := range items {
		docs[i] = item
	}

	// Execute bulk insert
	_, err := m.Collection.InsertMany(ctx, docs)
	if err != nil {
		return storeErrors.New(ErrBulkOpFailed).
			WithDetail("operation", "insert").
			WithDetail("count", len(items)).
			WithDetail("collection", m.Collection.Name()).
			WithCause(err)
	}

	return nil
}

// BulkUpdate modifies multiple documents in a single operation
func (m *TypedMongo[T]) BulkUpdate(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	// Create bulk write models
	models := make([]mongo.WriteModel, 0, len(items))
	for _, item := range items {
		// Extract ID field value using reflection
		idValue := extractIDValue(item, m.IDField)
		if idValue == nil {
			continue
		}

		model := mongo.NewUpdateOneModel().
			SetFilter(bson.M{m.IDField: idValue}).
			SetUpdate(bson.M{"$set": item})

		models = append(models, model)
	}

	if len(models) == 0 {
		return nil
	}

	// Execute bulk write
	_, err := m.Collection.BulkWrite(ctx, models)
	if err != nil {
		return storeErrors.New(ErrBulkOpFailed).
			WithDetail("operation", "update").
			WithDetail("count", len(models)).
			WithDetail("collection", m.Collection.Name()).
			WithCause(err)
	}

	return nil
}

// BulkDelete removes multiple documents in a single operation
func (m *TypedMongo[T]) BulkDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Convert string IDs to appropriate type
	idValues := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		if m.IDField == "_id" {
			// Convert to ObjectID if using MongoDB's native IDs
			objectID, err := primitive.ObjectIDFromHex(id)
			if err != nil {
				continue // Skip invalid IDs
			}
			idValues = append(idValues, objectID)
		} else {
			idValues = append(idValues, id)
		}
	}

	if len(idValues) == 0 {
		return nil
	}

	// Execute bulk delete
	_, err := m.Collection.DeleteMany(ctx, bson.M{m.IDField: bson.M{"$in": idValues}})
	if err != nil {
		return storeErrors.New(ErrBulkOpFailed).
			WithDetail("operation", "delete").
			WithDetail("count", len(ids)).
			WithDetail("collection", m.Collection.Name()).
			WithCause(err)
	}

	return nil
}

// Search performs a full-text search
func (m *TypedMongo[T]) Search(ctx context.Context, query string, opts SearchOptions) ([]T, error) {
	var results []T

	// Create text search filter
	filter := bson.M{"$text": bson.M{"$search": query}}

	// Create find options
	findOpts := options.Find()

	// Set limit and skip
	if opts.Limit > 0 {
		findOpts.SetLimit(int64(opts.Limit))
	}
	if opts.Offset > 0 {
		findOpts.SetSkip(int64(opts.Offset))
	}

	// Set sort by text score
	findOpts.SetSort(bson.M{"score": bson.M{"$meta": "textScore"}})

	// Set projection to include score
	projection := bson.M{"score": bson.M{"$meta": "textScore"}}
	findOpts.SetProjection(projection)

	// Execute search
	cursor, err := m.Collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, storeErrors.New(ErrSearchFailed).
			WithDetail("query", query).
			WithDetail("collection", m.Collection.Name()).
			WithCause(err)
	}
	defer cursor.Close(ctx)

	// Decode results
	if err := cursor.All(ctx, &results); err != nil {
		return nil, storeErrors.New(ErrMongoDecodeFailed).
			WithDetail("query", query).
			WithDetail("collection", m.Collection.Name()).
			WithCause(err)
	}

	return results, nil
}

// Watch creates a change stream for real-time notifications
func (m *TypedMongo[T]) Watch(ctx context.Context, filter map[string]any) (<-chan ChangeEvent[T], error) {
	// Create pipeline for change stream
	pipeline := mongo.Pipeline{}

	// Add match stage if filter is provided
	if len(filter) > 0 {
		bsonFilter := convertMapToBson(filter)
		matchStage := bson.D{{Key: "$match", Value: bsonFilter}}
		pipeline = append(pipeline, matchStage)
	}

	// Create options with full document lookup
	opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)

	// Create change stream
	changeStream, err := m.Collection.Watch(ctx, pipeline, opts)
	if err != nil {
		return nil, err
	}

	// Create channel for change events
	eventChan := make(chan ChangeEvent[T], 100)

	// Start goroutine to process change events
	go func() {
		defer changeStream.Close(ctx)
		defer close(eventChan)

		for changeStream.Next(ctx) {
			var changeDoc struct {
				OperationType string    `bson:"operationType"`
				FullDocument  T         `bson:"fullDocument"`
				DocumentKey   bson.M    `bson:"documentKey"`
				ClusterTime   time.Time `bson:"clusterTime"`
			}

			if err := changeStream.Decode(&changeDoc); err != nil {
				continue
			}

			// Create change event
			event := ChangeEvent[T]{
				Operation: changeDoc.OperationType,
				Timestamp: changeDoc.ClusterTime,
			}

			// Set new value for insert and update operations
			if changeDoc.OperationType == "insert" || changeDoc.OperationType == "update" ||
				changeDoc.OperationType == "replace" {
				newVal := changeDoc.FullDocument
				event.NewValue = &newVal
			}

			// Send event to channel
			select {
			case eventChan <- event:
				// Event sent successfully
			case <-ctx.Done():
				return
			}
		}
	}()

	return eventChan, nil
}

// PaginateMongo is a helper function for MongoDB pagination
func PaginateMongo[T any](
	ctx context.Context,
	collection *mongo.Collection,
	opts PaginationOptions,
) (Paginated[T], error) {
	// Create filter from options
	filter := bson.M{}

	// Add filters from options
	for k, v := range opts.Filters {
		filter[k] = v
	}

	// 1. Count total documents
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return Paginated[T]{}, storeErrors.New(ErrMongoCountFailed).
			WithDetail("collection", collection.Name()).
			WithCause(err)
	}

	// 2. Set up options for pagination
	findOptions := options.Find()

	// Set skip and limit
	skip := int64((opts.Page - 1) * opts.PageSize)
	limit := int64(opts.PageSize)
	findOptions.SetSkip(skip)
	findOptions.SetLimit(limit)

	// Set sort
	if opts.OrderBy != "" {
		sortValue := 1
		if opts.Desc {
			sortValue = -1
		}
		findOptions.SetSort(bson.D{{Key: opts.OrderBy, Value: sortValue}})
	}

	// Set projection if fields are specified
	if len(opts.Fields) > 0 {
		projection := bson.M{}
		for _, field := range opts.Fields {
			projection[field] = 1
		}
		findOptions.SetProjection(projection)
	}

	// 3. Execute find with pagination
	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return Paginated[T]{}, storeErrors.New(ErrMongoFindFailed).
			WithDetail("collection", collection.Name()).
			WithDetail("filter", fmt.Sprintf("%v", filter)).
			WithCause(err)
	}
	defer cursor.Close(ctx)

	// 4. Decode the results
	var results []T
	if err := cursor.All(ctx, &results); err != nil {
		return Paginated[T]{}, storeErrors.New(ErrMongoDecodeFailed).
			WithDetail("collection", collection.Name()).
			WithCause(err)
	}

	// 5. Create and return the paginated result
	return NewPaginated(results, opts.Page, opts.PageSize, int(total)), nil
}

// Helper functions

// convertMapToBson converts a map to a BSON document
func convertMapToBson(m map[string]any) bson.M {
	bsonDoc := bson.M{}
	for k, v := range m {
		bsonDoc[k] = v
	}
	return bsonDoc
}

// extractIDValue extracts the ID field value from a struct using reflection
func extractIDValue(item interface{}, idField string) interface{} {
	// Implementation would use reflection to extract the ID field
	// For simplicity, this is a placeholder
	return nil
}

