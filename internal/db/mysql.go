package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLDriver implements the Driver interface for MySQL.
type MySQLDriver struct {
	db *sql.DB
}

// NewMySQLDriver creates a new MySQL driver.
func NewMySQLDriver() *MySQLDriver {
	return &MySQLDriver{}
}

func (d *MySQLDriver) Connect(config ConnectionConfig) error {
	dsn, err := buildMySQLDSN(config)
	if err != nil {
		return err
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	d.db = db
	return nil
}

// buildMySQLDSN constructs a go-sql-driver/mysql DSN from a ConnectionConfig.
// The driver expects the format: user:password@tcp(host:port)/dbname?params
func buildMySQLDSN(config ConnectionConfig) (string, error) {
	if config.URI != "" {
		// The URI is in the standard URL form mysql://user:pass@host:port/db
		// but go-sql-driver/mysql expects user:pass@tcp(host:port)/db.
		// Parse and convert.
		return convertURItoDSN(config.URI)
	}

	host := config.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := config.Port
	if port == 0 {
		port = 3306
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		config.User, config.Password, host, port, config.Database)
	return dsn, nil
}

// convertURItoDSN converts a mysql://user:pass@host:port/db URI into the
// go-sql-driver/mysql DSN format user:pass@tcp(host:port)/db.
func convertURItoDSN(uri string) (string, error) {
	// Strip the mysql:// scheme prefix
	raw := uri
	if strings.HasPrefix(raw, "mysql://") {
		raw = strings.TrimPrefix(raw, "mysql://")
	}

	// Split userinfo from the rest: userinfo@hostAndPath
	var userInfo, hostAndPath string
	if idx := strings.LastIndex(raw, "@"); idx >= 0 {
		userInfo = raw[:idx]
		hostAndPath = raw[idx+1:]
	} else {
		hostAndPath = raw
	}

	// Split host:port from /dbname?params
	var hostPort, dbAndParams string
	if idx := strings.Index(hostAndPath, "/"); idx >= 0 {
		hostPort = hostAndPath[:idx]
		dbAndParams = hostAndPath[idx+1:]
	} else {
		hostPort = hostAndPath
	}

	if hostPort == "" {
		hostPort = "127.0.0.1:3306"
	} else if !strings.Contains(hostPort, ":") {
		hostPort += ":3306"
	}

	// Build DSN
	dsn := ""
	if userInfo != "" {
		dsn = userInfo + "@"
	}
	dsn += fmt.Sprintf("tcp(%s)/%s", hostPort, dbAndParams)

	// Append parseTime if not already present
	if !strings.Contains(dsn, "parseTime") {
		if strings.Contains(dsn, "?") {
			dsn += "&parseTime=true"
		} else {
			dsn += "?parseTime=true"
		}
	}

	return dsn, nil
}

func (d *MySQLDriver) Disconnect() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *MySQLDriver) DatabaseType() string {
	return "MySQL"
}

func (d *MySQLDriver) GetDatabases() ([]string, error) {
	rows, err := d.db.Query("SHOW DATABASES")
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	defer rows.Close()

	systemDBs := map[string]bool{
		"information_schema": true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
	}

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if !systemDBs[name] {
			databases = append(databases, name)
		}
	}
	return databases, nil
}

func (d *MySQLDriver) GetSchemas(database string) ([]string, error) {
	// MySQL does not have schemas in the PostgreSQL sense.
	// Each database is its own namespace, so return the database name.
	if database == "" {
		return []string{"default"}, nil
	}
	return []string{database}, nil
}

func (d *MySQLDriver) GetTables(database, _ string) ([]Table, error) {
	query := `SELECT TABLE_NAME, TABLE_TYPE, TABLE_ROWS
		FROM information_schema.tables
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME`

	rows, err := d.db.Query(query, database)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var name, tableType string
		var rowCount sql.NullInt64
		if err := rows.Scan(&name, &tableType, &rowCount); err != nil {
			return nil, err
		}

		t := Table{
			Name:   name,
			Schema: database,
			Type:   normalizeTableType(tableType),
		}
		if rowCount.Valid {
			t.RowCount = rowCount.Int64
		} else {
			t.RowCount = -1
		}
		tables = append(tables, t)
	}
	return tables, nil
}

// normalizeTableType converts MySQL table types to the project's convention.
func normalizeTableType(mysqlType string) string {
	switch strings.ToUpper(mysqlType) {
	case "BASE TABLE":
		return "table"
	case "VIEW":
		return "view"
	case "SYSTEM VIEW":
		return "view"
	default:
		return strings.ToLower(mysqlType)
	}
}

func (d *MySQLDriver) GetColumns(database, _, table string) ([]Column, error) {
	query := `SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, COLUMN_DEFAULT, EXTRA
		FROM information_schema.columns
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`

	rows, err := d.db.Query(query, database, table)
	if err != nil {
		return nil, fmt.Errorf("failed to list columns: %w", err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var name, dataType, nullable, columnKey string
		var dflt sql.NullString
		var extra string
		if err := rows.Scan(&name, &dataType, &nullable, &columnKey, &dflt, &extra); err != nil {
			return nil, err
		}

		col := Column{
			Name:       name,
			DataType:   dataType,
			Nullable:   strings.EqualFold(nullable, "YES"),
			PrimaryKey: columnKey == "PRI",
			Extra:      extra,
		}
		if dflt.Valid {
			col.Default = dflt.String
		}
		columns = append(columns, col)
	}
	return columns, nil
}

func (d *MySQLDriver) GetIndexes(database, _, table string) ([]Index, error) {
	query := `SELECT INDEX_NAME, COLUMN_NAME, NON_UNIQUE, INDEX_TYPE
		FROM information_schema.statistics
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX`

	rows, err := d.db.Query(query, database, table)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer rows.Close()

	// Group columns by index name, preserving order.
	type indexInfo struct {
		name      string
		unique    bool
		indexType string
		columns   []string
	}
	indexMap := make(map[string]*indexInfo)
	var indexOrder []string

	for rows.Next() {
		var indexName, colName, idxType string
		var nonUnique int
		if err := rows.Scan(&indexName, &colName, &nonUnique, &idxType); err != nil {
			return nil, err
		}
		if info, ok := indexMap[indexName]; ok {
			info.columns = append(info.columns, colName)
		} else {
			indexMap[indexName] = &indexInfo{
				name:      indexName,
				unique:    nonUnique == 0,
				indexType: strings.ToLower(idxType),
				columns:   []string{colName},
			}
			indexOrder = append(indexOrder, indexName)
		}
	}

	var indexes []Index
	for _, name := range indexOrder {
		info := indexMap[name]
		indexes = append(indexes, Index{
			Name:    info.name,
			Columns: info.columns,
			Unique:  info.unique,
			Type:    info.indexType,
		})
	}
	return indexes, nil
}

func (d *MySQLDriver) Execute(query string) (*QueryResult, error) {
	start := time.Now()

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
				switch val := v.(type) {
				case []byte:
					row[i] = string(val)
				default:
					row[i] = fmt.Sprintf("%v", val)
				}
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

func (d *MySQLDriver) GetTablePreview(database, _, table string, limit int) (*QueryResult, error) {
	// Use backtick quoting for MySQL identifiers.
	query := fmt.Sprintf("SELECT * FROM `%s`.`%s` LIMIT %d", database, table, limit)
	return d.Execute(query)
}

func (d *MySQLDriver) ExplainQuery(query string) (string, error) {
	result, err := d.Execute("EXPLAIN " + query)
	if err != nil {
		return "", err
	}

	var lines []string
	// Build a header line from column names.
	lines = append(lines, strings.Join(result.Columns, "\t"))
	for _, row := range result.Rows {
		lines = append(lines, strings.Join(row, "\t"))
	}
	return strings.Join(lines, "\n"), nil
}
