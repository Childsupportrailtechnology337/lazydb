package db

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisDriver implements the Driver interface for Redis.
type RedisDriver struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisDriver creates a new Redis driver.
func NewRedisDriver() *RedisDriver {
	return &RedisDriver{
		ctx: context.Background(),
	}
}

func (d *RedisDriver) Connect(config ConnectionConfig) error {
	var opts *redis.Options

	if config.URI != "" {
		var err error
		opts, err = redis.ParseURL(config.URI)
		if err != nil {
			return fmt.Errorf("failed to parse Redis URI: %w", err)
		}
	} else {
		host := config.Host
		if host == "" {
			host = "localhost"
		}
		port := config.Port
		if port == 0 {
			port = 6379
		}

		db := 0
		if config.Database != "" {
			var err error
			db, err = strconv.Atoi(config.Database)
			if err != nil {
				db = 0
			}
		}

		opts = &redis.Options{
			Addr:     fmt.Sprintf("%s:%d", host, port),
			Password: config.Password,
			DB:       db,
		}
	}

	d.client = redis.NewClient(opts)

	if err := d.client.Ping(d.ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return nil
}

func (d *RedisDriver) Disconnect() error {
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}

func (d *RedisDriver) DatabaseType() string {
	return "Redis"
}

func (d *RedisDriver) GetDatabases() ([]string, error) {
	dbs := make([]string, 16)
	for i := 0; i < 16; i++ {
		dbs[i] = strconv.Itoa(i)
	}
	return dbs, nil
}

func (d *RedisDriver) GetSchemas(_ string) ([]string, error) {
	return []string{"keys"}, nil
}

func (d *RedisDriver) GetTables(_, _ string) ([]Table, error) {
	prefixes := make(map[string]int64)
	var cursor uint64

	for {
		keys, next, err := d.client.Scan(d.ctx, cursor, "*", 1000).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan keys: %w", err)
		}

		for _, key := range keys {
			prefix := key
			if idx := strings.Index(key, ":"); idx != -1 {
				prefix = key[:idx]
			}
			prefixes[prefix]++
		}

		cursor = next
		if cursor == 0 {
			break
		}
	}

	tables := make([]Table, 0, len(prefixes))
	for prefix, count := range prefixes {
		tables = append(tables, Table{
			Name:     prefix,
			Schema:   "keys",
			RowCount: count,
			Type:     "keyspace",
		})
	}

	sort.Slice(tables, func(i, j int) bool {
		return tables[i].Name < tables[j].Name
	})

	return tables, nil
}

func (d *RedisDriver) GetColumns(_, _, table string) ([]Column, error) {
	// Sample keys matching this prefix and report their Redis types.
	pattern := table + ":*"
	keys, _, err := d.client.Scan(d.ctx, 0, pattern, 100).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to scan keys: %w", err)
	}

	// If no keys matched with prefix pattern, try the exact key.
	if len(keys) == 0 {
		exists, err := d.client.Exists(d.ctx, table).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to check key existence: %w", err)
		}
		if exists > 0 {
			keys = []string{table}
		}
	}

	typeCounts := make(map[string]int)
	for _, key := range keys {
		t, err := d.client.Type(d.ctx, key).Result()
		if err != nil {
			continue
		}
		typeCounts[t]++
	}

	columns := make([]Column, 0, len(typeCounts))
	for t, count := range typeCounts {
		columns = append(columns, Column{
			Name:     t,
			DataType: t,
			Extra:    fmt.Sprintf("%d keys", count),
		})
	}

	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Name < columns[j].Name
	})

	return columns, nil
}

func (d *RedisDriver) GetIndexes(_, _, _ string) ([]Index, error) {
	return []Index{}, nil
}

func (d *RedisDriver) Execute(query string) (*QueryResult, error) {
	start := time.Now()

	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty command")
	}

	args := parseRedisCommand(query)
	if len(args) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmd := strings.ToUpper(args[0])

	// Convert []string to []interface{} for the Redis client.
	redisArgs := make([]interface{}, len(args))
	for i, a := range args {
		redisArgs[i] = a
	}

	rawCmd := redis.NewCmd(d.ctx, redisArgs...)
	if err := d.client.Process(d.ctx, rawCmd); err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}

	result := &QueryResult{
		Duration: time.Since(start),
	}

	val := rawCmd.Val()

	switch v := val.(type) {
	case string:
		result.Columns = []string{"result"}
		result.Rows = [][]string{{v}}
	case int64:
		result.Columns = []string{"result"}
		result.Rows = [][]string{{strconv.FormatInt(v, 10)}}
	case float64:
		result.Columns = []string{"result"}
		result.Rows = [][]string{{strconv.FormatFloat(v, 'f', -1, 64)}}
	case []interface{}:
		result = formatSliceResult(cmd, v, time.Since(start))
	case map[interface{}]interface{}:
		result.Columns = []string{"field", "value"}
		for mk, mv := range v {
			result.Rows = append(result.Rows, []string{
				fmt.Sprintf("%v", mk),
				fmt.Sprintf("%v", mv),
			})
		}
	case nil:
		result.Columns = []string{"result"}
		result.Rows = [][]string{{"(nil)"}}
	default:
		result.Columns = []string{"result"}
		result.Rows = [][]string{{fmt.Sprintf("%v", v)}}
	}

	result.RowCount = len(result.Rows)
	if result.RowCount == 0 {
		result.Message = "OK"
	}

	return result, nil
}

func (d *RedisDriver) GetTablePreview(_, _, table string, limit int) (*QueryResult, error) {
	start := time.Now()

	// Try prefix pattern first, then exact key.
	pattern := table + ":*"
	keys, _, err := d.client.Scan(d.ctx, 0, pattern, int64(limit)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to scan keys: %w", err)
	}

	// If nothing matched with prefix, try exact key or wildcard on the table name itself.
	if len(keys) == 0 {
		exists, _ := d.client.Exists(d.ctx, table).Result()
		if exists > 0 {
			keys = []string{table}
		} else {
			keys, _, err = d.client.Scan(d.ctx, 0, table+"*", int64(limit)).Result()
			if err != nil {
				return nil, fmt.Errorf("failed to scan keys: %w", err)
			}
		}
	}

	if limit > 0 && len(keys) > limit {
		keys = keys[:limit]
	}

	result := &QueryResult{
		Columns: []string{"key", "type", "value"},
	}

	for _, key := range keys {
		t, err := d.client.Type(d.ctx, key).Result()
		if err != nil {
			continue
		}

		val := d.previewValue(key, t)
		result.Rows = append(result.Rows, []string{key, t, val})
	}

	result.RowCount = len(result.Rows)
	result.Duration = time.Since(start)

	return result, nil
}

func (d *RedisDriver) ExplainQuery(query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("empty command")
	}

	args := parseRedisCommand(query)
	if len(args) == 0 {
		return "", fmt.Errorf("empty command")
	}

	cmd := strings.ToUpper(args[0])
	remaining := args[1:]

	var info strings.Builder
	info.WriteString(fmt.Sprintf("Command: %s\n", cmd))
	if len(remaining) > 0 {
		info.WriteString(fmt.Sprintf("Arguments: %s\n", strings.Join(remaining, " ")))
	}

	switch cmd {
	case "GET":
		info.WriteString("Operation: Retrieve the string value of a key\n")
		info.WriteString("Time complexity: O(1)\n")
	case "SET":
		info.WriteString("Operation: Set the string value of a key\n")
		info.WriteString("Time complexity: O(1)\n")
	case "HGETALL":
		info.WriteString("Operation: Get all fields and values in a hash\n")
		info.WriteString("Time complexity: O(N) where N is the number of fields\n")
	case "LRANGE":
		info.WriteString("Operation: Get a range of elements from a list\n")
		info.WriteString("Time complexity: O(S+N) where S is start offset and N is number of elements\n")
	case "SMEMBERS":
		info.WriteString("Operation: Get all members of a set\n")
		info.WriteString("Time complexity: O(N) where N is the set cardinality\n")
	case "ZRANGE":
		info.WriteString("Operation: Get a range of members from a sorted set\n")
		info.WriteString("Time complexity: O(log(N)+M) where N is the number of elements and M is the number returned\n")
	case "KEYS":
		info.WriteString("Operation: Find all keys matching a pattern\n")
		info.WriteString("Time complexity: O(N) where N is the number of keys in the database\n")
		info.WriteString("Warning: KEYS should not be used in production; use SCAN instead\n")
	case "DEL":
		info.WriteString("Operation: Delete one or more keys\n")
		info.WriteString("Time complexity: O(N) where N is the number of keys removed\n")
	case "TTL":
		info.WriteString("Operation: Get the time to live for a key in seconds\n")
		info.WriteString("Time complexity: O(1)\n")
	case "TYPE":
		info.WriteString("Operation: Determine the type stored at a key\n")
		info.WriteString("Time complexity: O(1)\n")
	case "INFO":
		info.WriteString("Operation: Get information and statistics about the server\n")
		info.WriteString("Time complexity: O(1)\n")
	case "DBSIZE":
		info.WriteString("Operation: Return the number of keys in the selected database\n")
		info.WriteString("Time complexity: O(1)\n")
	case "SCAN":
		info.WriteString("Operation: Incrementally iterate over keys\n")
		info.WriteString("Time complexity: O(1) per call, O(N) for a full iteration\n")
	default:
		info.WriteString(fmt.Sprintf("Operation: Execute Redis command %s\n", cmd))
	}

	return info.String(), nil
}

// previewValue retrieves a human-readable preview of a key's value based on its type.
func (d *RedisDriver) previewValue(key, keyType string) string {
	switch keyType {
	case "string":
		val, err := d.client.Get(d.ctx, key).Result()
		if err != nil {
			return "<error>"
		}
		if len(val) > 200 {
			return val[:200] + "..."
		}
		return val
	case "hash":
		val, err := d.client.HGetAll(d.ctx, key).Result()
		if err != nil {
			return "<error>"
		}
		pairs := make([]string, 0, len(val))
		for k, v := range val {
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
		}
		s := strings.Join(pairs, ", ")
		if len(s) > 200 {
			return s[:200] + "..."
		}
		return s
	case "list":
		val, err := d.client.LRange(d.ctx, key, 0, 9).Result()
		if err != nil {
			return "<error>"
		}
		s := strings.Join(val, ", ")
		if len(s) > 200 {
			return s[:200] + "..."
		}
		return s
	case "set":
		val, err := d.client.SMembers(d.ctx, key).Result()
		if err != nil {
			return "<error>"
		}
		s := strings.Join(val, ", ")
		if len(s) > 200 {
			return s[:200] + "..."
		}
		return s
	case "zset":
		val, err := d.client.ZRange(d.ctx, key, 0, 9).Result()
		if err != nil {
			return "<error>"
		}
		s := strings.Join(val, ", ")
		if len(s) > 200 {
			return s[:200] + "..."
		}
		return s
	default:
		return fmt.Sprintf("<%s>", keyType)
	}
}

// parseRedisCommand splits a Redis command string into arguments,
// respecting quoted strings.
func parseRedisCommand(input string) []string {
	var args []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case ch == ' ' && !inSingle && !inDouble:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// formatSliceResult formats a []interface{} result from Redis into a QueryResult.
func formatSliceResult(cmd string, items []interface{}, duration time.Duration) *QueryResult {
	result := &QueryResult{
		Duration: duration,
	}

	// HGETALL returns alternating field/value pairs.
	if cmd == "HGETALL" && len(items)%2 == 0 {
		result.Columns = []string{"field", "value"}
		for i := 0; i < len(items); i += 2 {
			result.Rows = append(result.Rows, []string{
				fmt.Sprintf("%v", items[i]),
				fmt.Sprintf("%v", items[i+1]),
			})
		}
		return result
	}

	// ZRANGE with WITHSCORES returns alternating member/score pairs.
	if cmd == "ZRANGE" && len(items)%2 == 0 && len(items) > 0 {
		result.Columns = []string{"member", "score"}
		for i := 0; i < len(items); i += 2 {
			result.Rows = append(result.Rows, []string{
				fmt.Sprintf("%v", items[i]),
				fmt.Sprintf("%v", items[i+1]),
			})
		}
		return result
	}

	// Default: numbered list.
	result.Columns = []string{"index", "value"}
	for i, item := range items {
		result.Rows = append(result.Rows, []string{
			strconv.Itoa(i),
			fmt.Sprintf("%v", item),
		})
	}

	return result
}
