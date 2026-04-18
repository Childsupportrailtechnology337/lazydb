package db

import (
	"fmt"
	"net/url"
	"strings"
)

// ParseConnectionString parses a connection string or file path into a ConnectionConfig.
func ParseConnectionString(s string) (ConnectionConfig, error) {
	s = strings.TrimSpace(s)

	// Check if it's a file path (SQLite)
	if strings.HasSuffix(s, ".db") || strings.HasSuffix(s, ".sqlite") ||
		strings.HasSuffix(s, ".sqlite3") || strings.HasPrefix(s, "./") ||
		strings.HasPrefix(s, "/") || strings.HasPrefix(s, "~") ||
		(len(s) > 1 && s[1] == ':') { // Windows path
		return ConnectionConfig{
			Type:     "sqlite",
			FilePath: s,
			Name:     s,
		}, nil
	}

	// Check for sqlite:// URI
	if strings.HasPrefix(s, "sqlite://") || strings.HasPrefix(s, "sqlite:///") {
		path := strings.TrimPrefix(s, "sqlite://")
		path = strings.TrimPrefix(path, "/")
		return ConnectionConfig{
			Type:     "sqlite",
			FilePath: path,
			Name:     path,
		}, nil
	}

	// Parse as URL
	u, err := url.Parse(s)
	if err != nil {
		return ConnectionConfig{}, fmt.Errorf("invalid connection string: %w", err)
	}

	config := ConnectionConfig{
		URI:  s,
		Host: u.Hostname(),
		Name: s,
	}

	if u.Port() != "" {
		fmt.Sscanf(u.Port(), "%d", &config.Port)
	}

	if u.User != nil {
		config.User = u.User.Username()
		config.Password, _ = u.User.Password()
	}

	config.Database = strings.TrimPrefix(u.Path, "/")

	switch u.Scheme {
	case "postgres", "postgresql":
		config.Type = "postgres"
		config.SSLMode = u.Query().Get("sslmode")
	case "mysql":
		config.Type = "mysql"
	case "mongodb", "mongodb+srv":
		config.Type = "mongodb"
	case "redis", "rediss":
		config.Type = "redis"
	default:
		return ConnectionConfig{}, fmt.Errorf("unsupported database type: %s", u.Scheme)
	}

	return config, nil
}

// NewDriver creates a new driver for the given database type.
func NewDriver(dbType string) (Driver, error) {
	switch dbType {
	case "sqlite":
		return NewSQLiteDriver(), nil
	case "postgres", "postgresql":
		return NewPostgresDriver(), nil
	case "mysql":
		return NewMySQLDriver(), nil
	case "mongodb":
		return NewMongoDriver(), nil
	case "redis":
		return NewRedisDriver(), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s (supported: sqlite, postgres, mysql, mongodb, redis)", dbType)
	}
}
