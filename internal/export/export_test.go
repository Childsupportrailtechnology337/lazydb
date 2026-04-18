package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
)

func sampleResult() *db.QueryResult {
	return &db.QueryResult{
		Columns: []string{"id", "name", "email", "active"},
		Rows: [][]string{
			{"1", "Alice", "alice@test.com", "1"},
			{"2", "Bob", "bob@test.com", "1"},
			{"3", "Carol", "carol@test.com", "0"},
		},
		RowCount: 3,
	}
}

func TestCSVExport(t *testing.T) {
	var buf bytes.Buffer
	err := CSV(&buf, sampleResult())
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 4 { // header + 3 rows
		t.Errorf("got %d lines, want 4", len(lines))
	}

	if lines[0] != "id,name,email,active" {
		t.Errorf("header = %q", lines[0])
	}
	if lines[1] != "1,Alice,alice@test.com,1" {
		t.Errorf("first row = %q", lines[1])
	}
}

func TestCSVExportEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := CSV(&buf, &db.QueryResult{
		Columns: []string{"id"},
		Rows:    [][]string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if !strings.Contains(output, "id") {
		t.Error("should still have header")
	}
}

func TestJSONExport(t *testing.T) {
	var buf bytes.Buffer
	err := JSON(&buf, sampleResult())
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, `"name": "Alice"`) {
		t.Errorf("missing Alice in JSON output")
	}
	if !strings.Contains(output, `"email": "bob@test.com"`) {
		t.Error("missing Bob's email")
	}
	// Should be valid JSON array
	if !strings.HasPrefix(strings.TrimSpace(output), "[") {
		t.Error("JSON should start with [")
	}
}

func TestSQLExport(t *testing.T) {
	var buf bytes.Buffer
	err := SQL(&buf, "users", sampleResult())
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 3 {
		t.Errorf("got %d lines, want 3", len(lines))
	}

	if !strings.HasPrefix(lines[0], "INSERT INTO users") {
		t.Errorf("first line = %q", lines[0])
	}
	if !strings.Contains(lines[0], "'Alice'") {
		t.Error("missing Alice value")
	}
}

func TestSQLExportNullHandling(t *testing.T) {
	result := &db.QueryResult{
		Columns: []string{"id", "value"},
		Rows: [][]string{
			{"1", "<NULL>"},
		},
	}

	var buf bytes.Buffer
	err := SQL(&buf, "test", result)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "NULL") {
		t.Error("should contain NULL")
	}
	if strings.Contains(output, "'<NULL>'") {
		t.Error("should not contain quoted <NULL>")
	}
}

func TestSQLExportEscaping(t *testing.T) {
	result := &db.QueryResult{
		Columns: []string{"name"},
		Rows: [][]string{
			{"O'Brien"},
		},
	}

	var buf bytes.Buffer
	err := SQL(&buf, "people", result)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "O''Brien") {
		t.Errorf("single quote not escaped: %s", output)
	}
}
