package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoDriver implements the Driver interface for MongoDB.
type MongoDriver struct {
	client   *mongo.Client
	database string // default database from connection config
}

// NewMongoDriver creates a new MongoDB driver.
func NewMongoDriver() *MongoDriver {
	return &MongoDriver{}
}

func (d *MongoDriver) Connect(config ConnectionConfig) error {
	uri := config.URI
	if uri == "" {
		host := config.Host
		if host == "" {
			host = "localhost"
		}
		port := config.Port
		if port == 0 {
			port = 27017
		}
		uri = fmt.Sprintf("mongodb://%s:%d", host, port)
		if config.User != "" {
			uri = fmt.Sprintf("mongodb://%s:%s@%s:%d",
				config.User, config.Password, host, port)
		}
		if config.Database != "" {
			uri += "/" + config.Database
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	d.client = client
	d.database = config.Database
	return nil
}

func (d *MongoDriver) Disconnect() error {
	if d.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return d.client.Disconnect(ctx)
	}
	return nil
}

func (d *MongoDriver) DatabaseType() string {
	return "MongoDB"
}

func (d *MongoDriver) GetDatabases() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	names, err := d.client.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	return names, nil
}

func (d *MongoDriver) GetSchemas(_ string) ([]string, error) {
	// MongoDB doesn't have schemas; return a single default schema.
	return []string{"default"}, nil
}

func (d *MongoDriver) GetTables(database, _ string) ([]Table, error) {
	dbName := d.resolveDatabase(database)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := d.client.Database(dbName)
	names, err := db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	var tables []Table
	for _, name := range names {
		count, err := db.Collection(name).EstimatedDocumentCount(ctx)
		if err != nil {
			count = -1
		}
		tables = append(tables, Table{
			Name:     name,
			Schema:   "default",
			RowCount: count,
			Type:     "collection",
		})
	}
	return tables, nil
}

func (d *MongoDriver) GetColumns(database, _, table string) ([]Column, error) {
	dbName := d.resolveDatabase(database)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.client.Database(dbName).Collection(table)

	// Sample up to 100 documents to infer the schema.
	cursor, err := coll.Find(ctx, bson.D{}, options.Find().SetLimit(100))
	if err != nil {
		return nil, fmt.Errorf("failed to sample collection: %w", err)
	}
	defer cursor.Close(ctx)

	// Track field names and their observed types.
	fieldTypes := make(map[string]string)
	fieldOrder := make(map[string]int)
	order := 0

	for cursor.Next(ctx) {
		var doc bson.D
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		for _, elem := range doc {
			if _, exists := fieldTypes[elem.Key]; !exists {
				fieldTypes[elem.Key] = bsonTypeString(elem.Value)
				fieldOrder[elem.Key] = order
				order++
			}
		}
	}

	var columns []Column
	for name, dataType := range fieldTypes {
		col := Column{
			Name:       name,
			DataType:   dataType,
			Nullable:   true,
			PrimaryKey: name == "_id",
		}
		columns = append(columns, col)
	}

	// Sort columns by the order they were first seen.
	sort.Slice(columns, func(i, j int) bool {
		return fieldOrder[columns[i].Name] < fieldOrder[columns[j].Name]
	})

	return columns, nil
}

func (d *MongoDriver) GetIndexes(database, _, table string) ([]Index, error) {
	dbName := d.resolveDatabase(database)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.client.Database(dbName).Collection(table)
	cursor, err := coll.Indexes().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer cursor.Close(ctx)

	var indexes []Index
	for cursor.Next(ctx) {
		var raw bson.M
		if err := cursor.Decode(&raw); err != nil {
			continue
		}

		name, _ := raw["name"].(string)

		var cols []string
		if keyDoc, ok := raw["key"].(bson.D); ok {
			for _, k := range keyDoc {
				cols = append(cols, k.Key)
			}
		}

		unique := false
		if u, ok := raw["unique"].(bool); ok {
			unique = u
		}

		indexes = append(indexes, Index{
			Name:    name,
			Columns: cols,
			Unique:  unique,
			Type:    "btree",
		})
	}

	return indexes, nil
}

// mongoCommand represents a structured MongoDB query.
type mongoCommand struct {
	Collection string      `json:"collection"`
	Operation  string      `json:"operation"`
	Filter     interface{} `json:"filter"`
	Update     interface{} `json:"update"`
	Sort       interface{} `json:"sort"`
	Limit      *int64      `json:"limit"`
}

func (d *MongoDriver) Execute(query string) (*QueryResult, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query = strings.TrimSpace(query)

	// Try to parse as a structured JSON command first.
	var cmd mongoCommand
	if err := json.Unmarshal([]byte(query), &cmd); err == nil && cmd.Collection != "" {
		result, err := d.executeCommand(ctx, cmd)
		if err != nil {
			return nil, err
		}
		result.Duration = time.Since(start)
		return result, nil
	}

	// Try to parse db.collection.operation(...) style queries.
	if strings.HasPrefix(query, "db.") {
		result, err := d.executeDotNotation(ctx, query)
		if err != nil {
			return nil, err
		}
		result.Duration = time.Since(start)
		return result, nil
	}

	// Treat as a raw JSON filter on the default database's first collection.
	var filter bson.D
	if err := bson.UnmarshalExtJSON([]byte(query), false, &filter); err != nil {
		return nil, fmt.Errorf("unsupported query format: %s", query)
	}

	return nil, fmt.Errorf("ambiguous query: provide a collection using {\"collection\": \"name\", \"operation\": \"find\", \"filter\": ...} or db.collection.find(...)")
}

func (d *MongoDriver) executeCommand(ctx context.Context, cmd mongoCommand) (*QueryResult, error) {
	dbName := d.resolveDatabase("")
	coll := d.client.Database(dbName).Collection(cmd.Collection)

	filterDoc, err := toBsonD(cmd.Filter)
	if err != nil {
		filterDoc = bson.D{}
	}

	switch strings.ToLower(cmd.Operation) {
	case "find", "":
		opts := options.Find()
		if cmd.Limit != nil {
			opts.SetLimit(*cmd.Limit)
		} else {
			opts.SetLimit(100) // default safety limit
		}
		if cmd.Sort != nil {
			sortDoc, err := toBsonD(cmd.Sort)
			if err == nil {
				opts.SetSort(sortDoc)
			}
		}
		cursor, findErr := coll.Find(ctx, filterDoc, opts)
		return d.cursorToResult(ctx, cursor, findErr)

	case "findone":
		var doc bson.M
		err := coll.FindOne(ctx, filterDoc).Decode(&doc)
		if err != nil {
			return nil, fmt.Errorf("findOne error: %w", err)
		}
		return d.docsToResult([]bson.M{doc}), nil

	case "count", "countdocuments":
		count, err := coll.CountDocuments(ctx, filterDoc)
		if err != nil {
			return nil, fmt.Errorf("count error: %w", err)
		}
		return &QueryResult{
			Columns:  []string{"count"},
			Rows:     [][]string{{fmt.Sprintf("%d", count)}},
			RowCount: 1,
			Message:  fmt.Sprintf("Count: %d", count),
		}, nil

	case "insertone":
		res, err := coll.InsertOne(ctx, filterDoc)
		if err != nil {
			return nil, fmt.Errorf("insertOne error: %w", err)
		}
		return &QueryResult{
			Columns:  []string{"insertedId"},
			Rows:     [][]string{{fmt.Sprintf("%v", res.InsertedID)}},
			RowCount: 1,
			Message:  "Document inserted",
		}, nil

	case "deleteone":
		res, err := coll.DeleteOne(ctx, filterDoc)
		if err != nil {
			return nil, fmt.Errorf("deleteOne error: %w", err)
		}
		return &QueryResult{
			Message: fmt.Sprintf("%d document(s) deleted", res.DeletedCount),
		}, nil

	case "deletemany":
		res, err := coll.DeleteMany(ctx, filterDoc)
		if err != nil {
			return nil, fmt.Errorf("deleteMany error: %w", err)
		}
		return &QueryResult{
			Message: fmt.Sprintf("%d document(s) deleted", res.DeletedCount),
		}, nil

	case "updateone":
		updateDoc, err := toBsonD(cmd.Update)
		if err != nil {
			return nil, fmt.Errorf("invalid update document: %w", err)
		}
		res, err := coll.UpdateOne(ctx, filterDoc, updateDoc)
		if err != nil {
			return nil, fmt.Errorf("updateOne error: %w", err)
		}
		return &QueryResult{
			Message: fmt.Sprintf("Matched: %d, Modified: %d", res.MatchedCount, res.ModifiedCount),
		}, nil

	case "updatemany":
		updateDoc, err := toBsonD(cmd.Update)
		if err != nil {
			return nil, fmt.Errorf("invalid update document: %w", err)
		}
		res, err := coll.UpdateMany(ctx, filterDoc, updateDoc)
		if err != nil {
			return nil, fmt.Errorf("updateMany error: %w", err)
		}
		return &QueryResult{
			Message: fmt.Sprintf("Matched: %d, Modified: %d", res.MatchedCount, res.ModifiedCount),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported operation: %s", cmd.Operation)
	}
}

func (d *MongoDriver) executeDotNotation(ctx context.Context, query string) (*QueryResult, error) {
	// Parse "db.collection.operation(...)" style queries.
	// Remove leading "db."
	rest := strings.TrimPrefix(query, "db.")

	// Find the operation part: collection.operation(...)
	dotIdx := strings.Index(rest, ".")
	if dotIdx < 0 {
		return nil, fmt.Errorf("invalid query format, expected db.collection.operation(...)")
	}
	collName := rest[:dotIdx]
	remainder := rest[dotIdx+1:]

	// Extract operation and arguments.
	parenIdx := strings.Index(remainder, "(")
	if parenIdx < 0 {
		return nil, fmt.Errorf("invalid query format, expected db.%s.operation(...)", collName)
	}
	operation := remainder[:parenIdx]

	// Extract the argument string between parentheses.
	argStr := remainder[parenIdx+1:]
	if strings.HasSuffix(argStr, ")") {
		argStr = argStr[:len(argStr)-1]
	}
	argStr = strings.TrimSpace(argStr)

	cmd := mongoCommand{
		Collection: collName,
		Operation:  operation,
	}

	if argStr != "" {
		// For operations like updateOne, there may be two arguments separated by comma
		// between two JSON objects: {filter}, {update}
		if strings.ToLower(operation) == "updateone" || strings.ToLower(operation) == "updatemany" {
			args := splitTopLevelJSON(argStr)
			if len(args) >= 1 {
				var filter interface{}
				if err := json.Unmarshal([]byte(args[0]), &filter); err == nil {
					cmd.Filter = filter
				}
			}
			if len(args) >= 2 {
				var update interface{}
				if err := json.Unmarshal([]byte(args[1]), &update); err == nil {
					cmd.Update = update
				}
			}
		} else {
			var filter interface{}
			if err := json.Unmarshal([]byte(argStr), &filter); err == nil {
				cmd.Filter = filter
			}
		}
	}

	return d.executeCommand(ctx, cmd)
}

func (d *MongoDriver) GetTablePreview(database, _, table string, limit int) (*QueryResult, error) {
	dbName := d.resolveDatabase(database)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.client.Database(dbName).Collection(table)
	opts := options.Find().SetLimit(int64(limit))

	cursor, findErr := coll.Find(ctx, bson.D{}, opts)
	result, err := d.cursorToResult(ctx, cursor, findErr)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (d *MongoDriver) ExplainQuery(query string) (string, error) {
	query = strings.TrimSpace(query)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to parse as structured command.
	var cmd mongoCommand
	if err := json.Unmarshal([]byte(query), &cmd); err != nil || cmd.Collection == "" {
		// Try dot notation.
		if strings.HasPrefix(query, "db.") {
			rest := strings.TrimPrefix(query, "db.")
			dotIdx := strings.Index(rest, ".")
			if dotIdx >= 0 {
				cmd.Collection = rest[:dotIdx]
				// Default to find with empty filter for explain.
				cmd.Operation = "find"
			}
		}
		if cmd.Collection == "" {
			return "", fmt.Errorf("cannot explain query: provide a structured command")
		}
	}

	dbName := d.resolveDatabase("")
	filterDoc, err := toBsonD(cmd.Filter)
	if err != nil {
		filterDoc = bson.D{}
	}

	explainCmd := bson.D{
		{Key: "explain", Value: bson.D{
			{Key: "find", Value: cmd.Collection},
			{Key: "filter", Value: filterDoc},
		}},
		{Key: "verbosity", Value: "executionStats"},
	}

	var result bson.M
	err = d.client.Database(dbName).RunCommand(ctx, explainCmd).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("explain error: %w", err)
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", result), nil
	}
	return string(jsonBytes), nil
}

// resolveDatabase returns the database name to use, preferring the argument
// over the default from the connection config.
func (d *MongoDriver) resolveDatabase(database string) string {
	if database != "" {
		return database
	}
	if d.database != "" {
		return d.database
	}
	return "test"
}

// cursorToResult converts a mongo cursor result into a QueryResult.
func (d *MongoDriver) cursorToResult(ctx context.Context, cursor *mongo.Cursor, err error) (*QueryResult, error) {
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to read results: %w", err)
	}

	return d.docsToResult(docs), nil
}

// docsToResult converts a slice of BSON documents into a QueryResult with
// consistent column ordering.
func (d *MongoDriver) docsToResult(docs []bson.M) *QueryResult {
	if len(docs) == 0 {
		return &QueryResult{
			Message: "No documents found",
		}
	}

	// Collect all unique keys across documents, preserving first-seen order.
	colSet := make(map[string]bool)
	var columns []string
	for _, doc := range docs {
		// Sort keys within each doc for deterministic ordering.
		keys := make([]string, 0, len(doc))
		for k := range doc {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !colSet[k] {
				colSet[k] = true
				columns = append(columns, k)
			}
		}
	}

	// Ensure _id is first if present.
	for i, c := range columns {
		if c == "_id" && i != 0 {
			columns = append(columns[:i], columns[i+1:]...)
			columns = append([]string{"_id"}, columns...)
			break
		}
	}

	var rows [][]string
	for _, doc := range docs {
		row := make([]string, len(columns))
		for i, col := range columns {
			val, ok := doc[col]
			if !ok || val == nil {
				row[i] = "<NULL>"
			} else {
				row[i] = bsonValueToString(val)
			}
		}
		rows = append(rows, row)
	}

	return &QueryResult{
		Columns:  columns,
		Rows:     rows,
		RowCount: len(rows),
	}
}

// bsonValueToString converts a BSON value to a human-readable string.
func bsonValueToString(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return "<NULL>"
	case string:
		return val
	case int32:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case bson.ObjectID:
		return val.Hex()
	case bson.D:
		b, err := bson.MarshalExtJSON(val, false, false)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	case bson.A:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	case bson.M:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}

// bsonTypeString returns a human-readable type name for a BSON value.
func bsonTypeString(v interface{}) string {
	switch v.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case int32:
		return "int32"
	case int64:
		return "int64"
	case float64:
		return "double"
	case bool:
		return "bool"
	case bson.ObjectID:
		return "objectId"
	case bson.D, bson.M:
		return "object"
	case bson.A:
		return "array"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// toBsonD converts an interface{} (typically from JSON unmarshal) to a bson.D.
func toBsonD(v interface{}) (bson.D, error) {
	if v == nil {
		return bson.D{}, nil
	}

	// If it's already bson.D, return it directly.
	if d, ok := v.(bson.D); ok {
		return d, nil
	}

	// Marshal to JSON, then unmarshal to bson.D via extended JSON.
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	var doc bson.D
	if err := bson.UnmarshalExtJSON(jsonBytes, false, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to BSON: %w", err)
	}
	return doc, nil
}

// splitTopLevelJSON splits a string containing multiple top-level JSON objects
// separated by commas, respecting brace nesting.
func splitTopLevelJSON(s string) []string {
	var parts []string
	depth := 0
	start := 0

	for i, ch := range s {
		switch ch {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		case ',':
			if depth == 0 {
				part := strings.TrimSpace(s[start:i])
				if part != "" {
					parts = append(parts, part)
				}
				start = i + 1
			}
		}
	}
	last := strings.TrimSpace(s[start:])
	if last != "" {
		parts = append(parts, last)
	}
	return parts
}
