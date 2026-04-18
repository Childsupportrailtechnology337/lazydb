package db

import "time"

// ConnectionConfig holds database connection parameters.
type ConnectionConfig struct {
	Type     string `yaml:"type"`
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	FilePath string `yaml:"file_path"` // For SQLite
	SSLMode  string `yaml:"ssl_mode"`
	URI      string `yaml:"uri"` // Raw connection string

	// SSH tunnel fields
	SSHHost     string `yaml:"ssh_host"`
	SSHPort     string `yaml:"ssh_port"`
	SSHUser     string `yaml:"ssh_user"`
	SSHPassword string `yaml:"ssh_password"`
	SSHKeyPath  string `yaml:"ssh_key_path"`
}

// Table represents a database table or collection.
type Table struct {
	Name     string
	Schema   string
	RowCount int64
	Type     string // "table", "view", "collection"
}

// Column represents a column in a table.
type Column struct {
	Name       string
	DataType   string
	Nullable   bool
	PrimaryKey bool
	Default    string
	Extra      string // auto_increment, etc.
}

// Index represents a database index.
type Index struct {
	Name    string
	Columns []string
	Unique  bool
	Type    string // btree, hash, etc.
}

// QueryResult holds the result of a query execution.
type QueryResult struct {
	Columns  []string
	Rows     [][]string
	RowCount int
	Duration time.Duration
	Message  string // For non-SELECT queries (e.g., "5 rows affected")
}

// Driver is the interface that all database drivers must implement.
type Driver interface {
	Connect(config ConnectionConfig) error
	Disconnect() error
	GetDatabases() ([]string, error)
	GetSchemas(database string) ([]string, error)
	GetTables(database, schema string) ([]Table, error)
	GetColumns(database, schema, table string) ([]Column, error)
	GetIndexes(database, schema, table string) ([]Index, error)
	Execute(query string) (*QueryResult, error)
	GetTablePreview(database, schema, table string, limit int) (*QueryResult, error)
	ExplainQuery(query string) (string, error)
	DatabaseType() string
}
