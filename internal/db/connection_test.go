package db

import (
	"testing"
)

func TestParseConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
		wantErr  bool
	}{
		{"sqlite file .db", "mydb.db", "sqlite", false},
		{"sqlite file .sqlite", "data.sqlite", "sqlite", false},
		{"sqlite file .sqlite3", "data.sqlite3", "sqlite", false},
		{"sqlite relative path", "./test.db", "sqlite", false},
		{"sqlite absolute unix", "/tmp/test.db", "sqlite", false},
		{"sqlite uri", "sqlite:///path/to/db", "sqlite", false},
		{"postgres uri", "postgres://user:pass@localhost:5432/mydb", "postgres", false},
		{"postgresql uri", "postgresql://user:pass@host/db", "postgres", false},
		{"postgres with sslmode", "postgres://u:p@h/d?sslmode=disable", "postgres", false},
		{"mysql uri", "mysql://user:pass@localhost:3306/mydb", "mysql", false},
		{"mongodb uri", "mongodb://localhost:27017/mydb", "mongodb", false},
		{"mongodb+srv", "mongodb+srv://user:pass@cluster.mongodb.net/db", "mongodb", false},
		{"redis uri", "redis://localhost:6379", "redis", false},
		{"rediss uri", "rediss://localhost:6380", "redis", false},
		{"unsupported scheme", "ftp://host/path", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseConnectionString(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if cfg.Type != tt.wantType {
				t.Errorf("type = %q, want %q", cfg.Type, tt.wantType)
			}
		})
	}
}

func TestParsePostgresDetails(t *testing.T) {
	cfg, err := ParseConnectionString("postgres://admin:secret@db.example.com:5433/production?sslmode=require")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "db.example.com" {
		t.Errorf("host = %q, want db.example.com", cfg.Host)
	}
	if cfg.Port != 5433 {
		t.Errorf("port = %d, want 5433", cfg.Port)
	}
	if cfg.User != "admin" {
		t.Errorf("user = %q, want admin", cfg.User)
	}
	if cfg.Password != "secret" {
		t.Errorf("password = %q, want secret", cfg.Password)
	}
	if cfg.Database != "production" {
		t.Errorf("database = %q, want production", cfg.Database)
	}
	if cfg.SSLMode != "require" {
		t.Errorf("sslmode = %q, want require", cfg.SSLMode)
	}
}

func TestNewDriver(t *testing.T) {
	tests := []struct {
		dbType  string
		wantErr bool
	}{
		{"sqlite", false},
		{"postgres", false},
		{"postgresql", false},
		{"mysql", false},
		{"mongodb", false},
		{"redis", false},
		{"cassandra", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			_, err := NewDriver(tt.dbType)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
