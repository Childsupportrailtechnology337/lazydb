package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteDriver implements the Driver interface for SQLite.
type SQLiteDriver struct {
	db       *sql.DB
	filePath string
}

// NewSQLiteDriver creates a new SQLite driver.
func NewSQLiteDriver() *SQLiteDriver {
	return &SQLiteDriver{}
}

func (d *SQLiteDriver) Connect(config ConnectionConfig) error {
	path := config.FilePath
	if path == "" {
		path = config.Database
	}
	if path == "" {
		return fmt.Errorf("no SQLite file path provided")
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to connect to SQLite database: %w", err)
	}

	d.db = db
	d.filePath = path
	return nil
}

func (d *SQLiteDriver) Disconnect() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *SQLiteDriver) DatabaseType() string {
	return "SQLite"
}

func (d *SQLiteDriver) GetDatabases() ([]string, error) {
	// SQLite doesn't have multiple databases in the traditional sense
	return nil, fmt.Errorf("not supported")
}

func (d *SQLiteDriver) GetSchemas(_ string) ([]string, error) {
	return []string{"main"}, nil
}

func (d *SQLiteDriver) GetTables(_, schema string) ([]Table, error) {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		// Get row count
		var count int64
		countRow := d.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %q", name))
		if err := countRow.Scan(&count); err != nil {
			count = -1
		}

		tables = append(tables, Table{
			Name:     name,
			Schema:   schema,
			RowCount: count,
			Type:     "table",
		})
	}
	return tables, nil
}

func (d *SQLiteDriver) GetColumns(_, _, table string) ([]Column, error) {
	rows, err := d.db.Query(fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dflt, &pk); err != nil {
			return nil, err
		}
		col := Column{
			Name:       name,
			DataType:   dataType,
			Nullable:   notNull == 0,
			PrimaryKey: pk > 0,
		}
		if dflt.Valid {
			col.Default = dflt.String
		}
		columns = append(columns, col)
	}
	return columns, nil
}

func (d *SQLiteDriver) GetIndexes(_, _, table string) ([]Index, error) {
	rows, err := d.db.Query(fmt.Sprintf("PRAGMA index_list(%q)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []Index
	for rows.Next() {
		var seq int
		var name, origin string
		var unique, partial int
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, err
		}

		// Get index columns
		colRows, err := d.db.Query(fmt.Sprintf("PRAGMA index_info(%q)", name))
		if err != nil {
			continue
		}
		var cols []string
		for colRows.Next() {
			var seqno, cid int
			var colName string
			if err := colRows.Scan(&seqno, &cid, &colName); err != nil {
				continue
			}
			cols = append(cols, colName)
		}
		colRows.Close()

		indexes = append(indexes, Index{
			Name:    name,
			Columns: cols,
			Unique:  unique == 1,
		})
	}
	return indexes, nil
}

func (d *SQLiteDriver) Execute(query string) (*QueryResult, error) {
	start := time.Now()

	// Try as a query first (SELECT, PRAGMA, etc.)
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := &QueryResult{
		Columns: cols,
	}

	values := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range values {
		ptrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make([]string, len(cols))
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

	if len(cols) == 0 {
		result.Message = "Query executed successfully"
	}

	return result, nil
}

func (d *SQLiteDriver) GetTablePreview(_, _, table string, limit int) (*QueryResult, error) {
	return d.Execute(fmt.Sprintf("SELECT * FROM %q LIMIT %d", table, limit))
}

func (d *SQLiteDriver) ExplainQuery(query string) (string, error) {
	result, err := d.Execute("EXPLAIN QUERY PLAN " + query)
	if err != nil {
		return "", err
	}
	var lines []string
	for _, row := range result.Rows {
		line := ""
		for _, cell := range row {
			line += cell + " "
		}
		lines = append(lines, line)
	}
	return fmt.Sprintf("%s", lines), nil
}
