package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// PostgresDriver implements the Driver interface for PostgreSQL.
type PostgresDriver struct {
	conn    *pgx.Conn
	connStr string
	version string
}

// NewPostgresDriver creates a new PostgreSQL driver.
func NewPostgresDriver() *PostgresDriver {
	return &PostgresDriver{}
}

func (d *PostgresDriver) Connect(config ConnectionConfig) error {
	connStr := config.URI
	if connStr == "" {
		host := config.Host
		if host == "" {
			host = "localhost"
		}
		port := config.Port
		if port == 0 {
			port = 5432
		}
		sslMode := config.SSLMode
		if sslMode == "" {
			sslMode = "prefer"
		}
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			config.User, config.Password, host, port, config.Database, sslMode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Get version
	var version string
	err = conn.QueryRow(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		version = "unknown"
	}

	d.conn = conn
	d.connStr = connStr
	d.version = version
	return nil
}

func (d *PostgresDriver) Disconnect() error {
	if d.conn != nil {
		return d.conn.Close(context.Background())
	}
	return nil
}

func (d *PostgresDriver) DatabaseType() string {
	return "PostgreSQL"
}

func (d *PostgresDriver) GetDatabases() ([]string, error) {
	ctx := context.Background()
	rows, err := d.conn.Query(ctx, `
		SELECT datname FROM pg_database
		WHERE datistemplate = false
		ORDER BY datname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dbs []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		dbs = append(dbs, name)
	}
	return dbs, nil
}

func (d *PostgresDriver) GetSchemas(_ string) ([]string, error) {
	ctx := context.Background()
	rows, err := d.conn.Query(ctx, `
		SELECT schema_name FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY schema_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		schemas = append(schemas, name)
	}
	return schemas, nil
}

func (d *PostgresDriver) GetTables(_, schema string) ([]Table, error) {
	if schema == "" {
		schema = "public"
	}
	ctx := context.Background()
	rows, err := d.conn.Query(ctx, `
		SELECT t.table_name, t.table_type,
			COALESCE(s.n_live_tup, 0)
		FROM information_schema.tables t
		LEFT JOIN pg_stat_user_tables s ON s.relname = t.table_name AND s.schemaname = t.table_schema
		WHERE t.table_schema = $1
		ORDER BY t.table_name`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var name, tableType string
		var rowCount int64
		if err := rows.Scan(&name, &tableType, &rowCount); err != nil {
			return nil, err
		}
		typ := "table"
		if tableType == "VIEW" {
			typ = "view"
		}
		tables = append(tables, Table{
			Name:     name,
			Schema:   schema,
			RowCount: rowCount,
			Type:     typ,
		})
	}
	return tables, nil
}

func (d *PostgresDriver) GetColumns(_, schema, table string) ([]Column, error) {
	if schema == "" {
		schema = "public"
	}
	ctx := context.Background()
	rows, err := d.conn.Query(ctx, `
		SELECT c.column_name, c.data_type, c.is_nullable, c.column_default,
			CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as is_pk
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku ON tc.constraint_name = ku.constraint_name
			WHERE tc.constraint_type = 'PRIMARY KEY'
				AND tc.table_schema = $1 AND tc.table_name = $2
		) pk ON pk.column_name = c.column_name
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var name, dataType, nullable string
		var dflt *string
		var isPK bool
		if err := rows.Scan(&name, &dataType, &nullable, &dflt, &isPK); err != nil {
			return nil, err
		}
		col := Column{
			Name:       name,
			DataType:   dataType,
			Nullable:   nullable == "YES",
			PrimaryKey: isPK,
		}
		if dflt != nil {
			col.Default = *dflt
		}
		columns = append(columns, col)
	}
	return columns, nil
}

func (d *PostgresDriver) GetIndexes(_, schema, table string) ([]Index, error) {
	if schema == "" {
		schema = "public"
	}
	ctx := context.Background()
	rows, err := d.conn.Query(ctx, `
		SELECT i.relname, ix.indisunique, am.amname,
			array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum))
		FROM pg_index ix
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_am am ON am.oid = i.relam
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = $1 AND t.relname = $2
		GROUP BY i.relname, ix.indisunique, am.amname
		ORDER BY i.relname`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []Index
	for rows.Next() {
		var name, amName string
		var unique bool
		var cols []string
		if err := rows.Scan(&name, &unique, &amName, &cols); err != nil {
			return nil, err
		}
		indexes = append(indexes, Index{
			Name:    name,
			Columns: cols,
			Unique:  unique,
			Type:    amName,
		})
	}
	return indexes, nil
}

func (d *PostgresDriver) Execute(query string) (*QueryResult, error) {
	ctx := context.Background()
	start := time.Now()

	rows, err := d.conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	descs := rows.FieldDescriptions()
	result := &QueryResult{}

	for _, desc := range descs {
		result.Columns = append(result.Columns, string(desc.Name))
	}

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := make([]string, len(values))
		for i, v := range values {
			if v == nil {
				row[i] = "<NULL>"
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		result.Rows = append(result.Rows, row)
	}

	result.RowCount = len(result.Rows)
	result.Duration = time.Since(start)

	if len(result.Columns) == 0 {
		result.Message = "Query executed successfully"
	}

	return result, nil
}

func (d *PostgresDriver) GetTablePreview(_, schema, table string, limit int) (*QueryResult, error) {
	if schema == "" {
		schema = "public"
	}
	return d.Execute(fmt.Sprintf("SELECT * FROM %q.%q LIMIT %d", schema, table, limit))
}

func (d *PostgresDriver) ExplainQuery(query string) (string, error) {
	result, err := d.Execute("EXPLAIN ANALYZE " + query)
	if err != nil {
		return "", err
	}
	var lines []string
	for _, row := range result.Rows {
		if len(row) > 0 {
			lines = append(lines, row[0])
		}
	}
	output := ""
	for _, l := range lines {
		output += l + "\n"
	}
	return output, nil
}
