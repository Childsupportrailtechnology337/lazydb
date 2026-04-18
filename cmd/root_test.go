package cmd

import (
	"testing"
)

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"single", "SELECT 1", 1},
		{"two with semicolons", "SELECT 1; SELECT 2;", 2},
		{"trailing semicolon", "SELECT 1;", 1},
		{"empty between", "SELECT 1;; SELECT 2", 2},
		{"with string containing semicolon", "SELECT 'hello;world'", 1},
		{"with double-quoted semicolon", `SELECT "col;name" FROM t`, 1},
		{"multiline", "SELECT\n  *\nFROM users;\nSELECT 1;", 2},
		{"with comment", "-- this is a comment\nSELECT 1;", 1},
		{"comment between", "SELECT 1;\n-- comment\nSELECT 2;", 2},
		{"empty input", "", 0},
		{"only whitespace", "  \n\n  ", 0},
		{"no trailing semicolon", "SELECT 1\nSELECT 2", 1}, // treated as single statement without ;
		{"insert with quotes", "INSERT INTO t VALUES ('it''s a test');", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmts := splitStatements(tt.input)
			if len(stmts) != tt.want {
				t.Errorf("got %d statements, want %d\n  stmts: %v", len(stmts), tt.want, stmts)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hello", 10) != "hello" {
		t.Error("short string should not be truncated")
	}
	if truncate("hello world", 5) != "hello..." {
		t.Errorf("truncate = %q", truncate("hello world", 5))
	}
}
