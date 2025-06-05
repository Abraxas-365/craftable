package storexmongo

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/storex"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoRepository is a MongoDB implementation of Repository
type MongoRepository[T any] struct {
	collection *mongo.Collection
	idField    string
}

// NewMongoRepository creates a new MongoDB repository
func NewMongoRepository[T any](collection *mongo.Collection, idField string) *MongoRepository[T] {
	if idField == "" {
		idField = "_id"
	}
	return &MongoRepository[T]{
		collection: collection,
		idField:    idField,
	}
}

// Create adds a new entity to the database
func (r *MongoRepository[T]) Create(ctx context.Context, item T) (T, error) {
	var empty T

	// Handle ID generation if using ObjectID
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Check for auto-generation of ID
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("bson")
		if tag == r.idField || strings.HasPrefix(tag, r.idField+",") {
			// If ID field is empty and type is string, generate ObjectID
			if isEmptyValue(v.Field(i)) && v.Field(i).Type().Kind() == reflect.String {
				idStr := primitive.NewObjectID().Hex()
				v.Field(i).SetString(idStr)
			}
		}
	}

	result, err := r.collection.InsertOne(ctx, item)
	if err != nil {
		return empty, storex.StoreErrors.NewWithCause(storex.ErrCreateFailed, err)
	}

	// For auto-generated IDs, we need to fetch the item
	if r.idField == "_id" && result.InsertedID != nil {
		filter := bson.M{"_id": result.InsertedID}
		err = r.collection.FindOne(ctx, filter).Decode(&item)
		if err != nil {
			return empty, storex.StoreErrors.NewWithCause(storex.ErrSQLQueryFailed, err)
		}
	}

	return item, nil
}

// FindByID retrieves an entity by its ID
func (r *MongoRepository[T]) FindByID(ctx context.Context, id string) (T, error) {
	var result T
	var empty T

	// Convert string ID to ObjectID if necessary
	var filter bson.M
	if r.idField == "_id" {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return empty, storex.StoreErrors.NewWithMessage(storex.ErrInvalidID, "Invalid ObjectID format")
		}
		filter = bson.M{"_id": objID}
	} else {
		filter = bson.M{r.idField: id}
	}

	err := r.collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return empty, storex.StoreErrors.NewWithMessage(storex.ErrRecordNotFound, "ID: "+id)
		}
		return empty, storex.StoreErrors.NewWithCause(storex.ErrSQLQueryFailed, err)
	}

	return result, nil
}

// FindOne retrieves a single entity that matches the filter
func (r *MongoRepository[T]) FindOne(ctx context.Context, filter map[string]any) (T, error) {
	var result T
	var empty T

	if len(filter) == 0 {
		return empty, storex.StoreErrors.NewWithMessage(storex.ErrInvalidQuery, "No filter provided")
	}

	// Convert map to BSON Document
	bsonFilter := bson.M{}
	for k, v := range filter {
		bsonFilter[k] = v
	}

	err := r.collection.FindOne(ctx, bsonFilter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return empty, storex.StoreErrors.NewWithMessage(storex.ErrRecordNotFound, "Filter not matched")
		}
		return empty, storex.StoreErrors.NewWithCause(storex.ErrSQLQueryFailed, err)
	}

	return result, nil
}

// Update modifies an existing entity
func (r *MongoRepository[T]) Update(ctx context.Context, id string, item T) (T, error) {
	var empty T

	// Prepare filter based on ID type
	var filter bson.M
	if r.idField == "_id" {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return empty, storex.StoreErrors.NewWithMessage(storex.ErrInvalidID, "Invalid ObjectID format")
		}
		filter = bson.M{"_id": objID}
	} else {
		filter = bson.M{r.idField: id}
	}

	// Update the document
	update := bson.M{"$set": item}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var result T
	err := r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return empty, storex.StoreErrors.NewWithMessage(storex.ErrRecordNotFound, "ID: "+id)
		}
		return empty, storex.StoreErrors.NewWithCause(storex.ErrUpdateFailed, err)
	}

	return result, nil
}

// Delete removes an entity from the store
func (r *MongoRepository[T]) Delete(ctx context.Context, id string) error {
	// Prepare filter based on ID type
	var filter bson.M
	if r.idField == "_id" {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return storex.StoreErrors.NewWithMessage(storex.ErrInvalidID, "Invalid ObjectID format")
		}
		filter = bson.M{"_id": objID}
	} else {
		filter = bson.M{r.idField: id}
	}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return storex.StoreErrors.NewWithCause(storex.ErrDeleteFailed, err)
	}

	if result.DeletedCount == 0 {
		return storex.StoreErrors.NewWithMessage(storex.ErrRecordNotFound, "ID: "+id)
	}

	return nil
}

// Paginate retrieves entities with pagination
func (r *MongoRepository[T]) Paginate(ctx context.Context, opts storex.PaginationOptions) (storex.Paginated[T], error) {
	// Convert filters to BSON
	filter := bson.M{}
	for k, v := range opts.Filters {
		filter[k] = v
	}

	// Build options
	findOptions := options.Find()

	// Sorting
	if opts.OrderBy != "" {
		sortDir := 1
		if opts.Desc {
			sortDir = -1
		}
		findOptions.SetSort(bson.D{{Key: opts.OrderBy, Value: sortDir}})
	}

	// Pagination
	offset := (opts.Page - 1) * opts.PageSize
	findOptions.SetSkip(int64(offset))
	findOptions.SetLimit(int64(opts.PageSize))

	// Field selection if specified
	if len(opts.Fields) > 0 {
		projection := bson.M{}
		for _, field := range opts.Fields {
			projection[field] = 1
		}
		findOptions.SetProjection(projection)
	}

	// Execute the query
	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return storex.Paginated[T]{}, storex.StoreErrors.NewWithCause(storex.ErrSQLQueryFailed, err)
	}
	defer cursor.Close(ctx)

	// Decode results
	var items []T
	if err = cursor.All(ctx, &items); err != nil {
		return storex.Paginated[T]{}, storex.StoreErrors.NewWithCause(storex.ErrSQLQueryFailed, err)
	}

	// Count total documents
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return storex.Paginated[T]{}, storex.StoreErrors.NewWithCause(storex.ErrSQLCountFailed, err)
	}

	return storex.NewPaginated(items, opts.Page, opts.PageSize, int(total)), nil
}

// MongoBulkOperator implements BulkOperator for MongoDB
type MongoBulkOperator[T any] struct {
	*MongoRepository[T]
}

// NewMongoBulkOperator creates a new MongoDB bulk operator
func NewMongoBulkOperator[T any](repo *MongoRepository[T]) *MongoBulkOperator[T] {
	return &MongoBulkOperator[T]{repo}
}

// BulkInsert adds multiple entities in a single operation
func (b *MongoBulkOperator[T]) BulkInsert(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	// Convert T items to interface{} for InsertMany
	documents := make([]interface{}, len(items))
	for i, item := range items {
		documents[i] = item
	}

	_, err := b.collection.InsertMany(ctx, documents)
	if err != nil {
		return storex.StoreErrors.NewWithCause(storex.ErrBulkOpFailed, err)
	}

	return nil
}

// BulkUpdate modifies multiple entities in a single operation
func (b *MongoBulkOperator[T]) BulkUpdate(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	// Create a bulk write operation
	models := make([]mongo.WriteModel, 0, len(items))

	for _, item := range items {
		v := reflect.ValueOf(item)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		// Find the ID field
		t := v.Type()
		var id interface{}
		var found bool

		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("bson")
			if tag == b.idField || strings.HasPrefix(tag, b.idField+",") {
				id = v.Field(i).Interface()
				found = true
				break
			}
		}

		if !found || id == nil {
			return storex.StoreErrors.NewWithMessage(storex.ErrInvalidID, "Missing ID for bulk update")
		}

		// Create update model
		filter := bson.M{b.idField: id}
		update := bson.M{"$set": item}

		model := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update)

		models = append(models, model)
	}

	_, err := b.collection.BulkWrite(ctx, models)
	if err != nil {
		return storex.StoreErrors.NewWithCause(storex.ErrBulkOpFailed, err)
	}

	return nil
}

// BulkDelete removes multiple entities in a single operation
func (b *MongoBulkOperator[T]) BulkDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Convert string IDs to ObjectIDs if necessary
	var filter bson.M
	if b.idField == "_id" {
		objectIDs := make([]primitive.ObjectID, 0, len(ids))
		for _, id := range ids {
			objID, err := primitive.ObjectIDFromHex(id)
			if err != nil {
				return storex.StoreErrors.NewWithMessage(storex.ErrInvalidID, "Invalid ObjectID format")
			}
			objectIDs = append(objectIDs, objID)
		}
		filter = bson.M{"_id": bson.M{"$in": objectIDs}}
	} else {
		filter = bson.M{b.idField: bson.M{"$in": ids}}
	}

	result, err := b.collection.DeleteMany(ctx, filter)
	if err != nil {
		return storex.StoreErrors.NewWithCause(storex.ErrBulkOpFailed, err)
	}

	if result.DeletedCount == 0 {
		return storex.StoreErrors.NewWithMessage(storex.ErrRecordNotFound, "No records matched IDs for deletion")
	}

	return nil
}

// MongoTxManager provides transaction support for MongoDB
type MongoTxManager struct {
	client *mongo.Client
}

// NewMongoTxManager creates a new MongoDB transaction manager
func NewMongoTxManager(client *mongo.Client) *MongoTxManager {
	return &MongoTxManager{client: client}
}

// WithTransaction executes operations within a transaction
func (tm *MongoTxManager) WithTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	session, err := tm.client.StartSession()
	if err != nil {
		return storex.StoreErrors.NewWithCause(storex.ErrTxBeginFailed, err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		err := fn(sessCtx)
		if err != nil {
			return nil, err // Error is already wrapped by the function
		}
		return nil, nil
	})

	if err != nil {
		return err
	}

	return nil
}

// MongoSearchable provides search for MongoDB
type MongoSearchable[T any] struct {
	*MongoRepository[T]
}

// NewMongoSearchable creates a new MongoDB searchable repository
func NewMongoSearchable[T any](repo *MongoRepository[T]) *MongoSearchable[T] {
	return &MongoSearchable[T]{repo}
}

// Search performs a text search using MongoDB's text index
func (s *MongoSearchable[T]) Search(ctx context.Context, query string, opts storex.SearchOptions) ([]T, error) {
	if len(opts.Fields) == 0 {
		return nil, storex.StoreErrors.NewWithMessage(storex.ErrInvalidQuery, "No search fields specified")
	}

	// Set default limit if not specified
	if opts.Limit <= 0 {
		opts.Limit = 25
	}

	// Create text search
	textSearchQuery := bson.M{
		"$text": bson.M{
			"$search": query,
		},
	}

	// Configure search options
	findOptions := options.Find()

	// Projection with score
	scoreProjection := bson.M{"score": bson.M{"$meta": "textScore"}}
	findOptions.SetProjection(scoreProjection)

	// Sort by score
	findOptions.SetSort(bson.D{{Key: "score", Value: bson.M{"$meta": "textScore"}}})

	// Limit and skip
	findOptions.SetLimit(int64(opts.Limit))
	findOptions.SetSkip(int64(opts.Offset))

	// Execute search
	cursor, err := s.collection.Find(ctx, textSearchQuery, findOptions)
	if err != nil {
		return nil, storex.StoreErrors.NewWithCause(storex.ErrSearchFailed, err)
	}
	defer cursor.Close(ctx)

	// Decode results
	var results []T
	if err = cursor.All(ctx, &results); err != nil {
		return nil, storex.StoreErrors.NewWithCause(storex.ErrSearchFailed, err)
	}

	return results, nil
}

// MongoChangeStream provides real-time notifications for MongoDB using change streams
type MongoChangeStream[T any] struct {
	*MongoRepository[T]
}

// NewMongoChangeStream creates a new MongoDB change stream
func NewMongoChangeStream[T any](repo *MongoRepository[T]) *MongoChangeStream[T] {
	return &MongoChangeStream[T]{repo}
}

// Watch creates a stream of change events
func (cs *MongoChangeStream[T]) Watch(ctx context.Context, filter map[string]any) (<-chan storex.ChangeEvent[T], error) {
	// Configure the change stream pipeline
	pipeline := mongo.Pipeline{}

	// Add match stage if filter is provided
	if len(filter) > 0 {
		matchStage := bson.D{{Key: "$match", Value: filter}}
		pipeline = append(pipeline, matchStage)
	}

	// Create options
	opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)

	// Set up the change stream
	stream, err := cs.collection.Watch(ctx, pipeline, opts)
	if err != nil {
		return nil, storex.StoreErrors.NewWithCause(storex.ErrConnectionFailed, err)
	}

	events := make(chan storex.ChangeEvent[T])

	go func() {
		defer close(events)
		defer stream.Close(ctx)

		for stream.Next(ctx) {
			var changeDoc struct {
				OperationType string              `bson:"operationType"`
				DocumentKey   bson.M              `bson:"documentKey"`
				FullDocument  T                   `bson:"fullDocument"`
				Timestamp     primitive.Timestamp `bson:"clusterTime"`
			}

			if err := stream.Decode(&changeDoc); err != nil {
				fmt.Printf("Error decoding change stream document: %v\n", err)
				continue
			}

			// Map MongoDB operation types to our operation types
			operation := ""
			switch changeDoc.OperationType {
			case "insert":
				operation = "insert"
			case "update", "replace":
				operation = "update"
			case "delete":
				operation = "delete"
			default:
				continue // Skip other operations
			}

			// Build the change event
			var oldValue, newValue *T
			if operation != "delete" {
				newValue = &changeDoc.FullDocument
			}

			event := storex.ChangeEvent[T]{
				Operation: operation,
				OldValue:  oldValue,
				NewValue:  newValue,
				Timestamp: time.Unix(int64(changeDoc.Timestamp.T), 0),
			}

			select {
			case events <- event:
				// Event sent successfully
			case <-ctx.Done():
				return
			}
		}

		if err := stream.Err(); err != nil {
			fmt.Printf("Change stream error: %v\n", err)
		}
	}()

	return events, nil
}

// Helper function to check if a value is empty
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
