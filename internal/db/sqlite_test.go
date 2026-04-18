package db

import (
	"os"
	"path/filepath"
	"testing"
)

func createTestSQLiteDB(t *testing.T) (*SQLiteDriver, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	driver := NewSQLiteDriver()
	err := driver.Connect(ConnectionConfig{FilePath: path})
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Create test schema
	stmts := []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT UNIQUE, active BOOLEAN DEFAULT 1)",
		"CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER REFERENCES users(id), title TEXT, body TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)",
		"CREATE INDEX idx_posts_user ON posts(user_id)",
		"INSERT INTO users (name, email, active) VALUES ('Alice', 'alice@test.com', 1)",
		"INSERT INTO users (name, email, active) VALUES ('Bob', 'bob@test.com', 1)",
		"INSERT INTO users (name, email, active) VALUES ('Carol', 'carol@test.com', 0)",
		"INSERT INTO posts (user_id, title, body) VALUES (1, 'Hello', 'World')",
		"INSERT INTO posts (user_id, title, body) VALUES (1, 'Second', 'Post')",
		"INSERT INTO posts (user_id, title, body) VALUES (2, 'Bobs Post', 'Content')",
	}
	for _, stmt := range stmts {
		if _, err := driver.Execute(stmt); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
	}
	return driver, path
}

func TestSQLiteConnect(t *testing.T) {
	driver, _ := createTestSQLiteDB(t)
	defer driver.Disconnect()

	if driver.DatabaseType() != "SQLite" {
		t.Errorf("DatabaseType = %q, want SQLite", driver.DatabaseType())
	}
}

func TestSQLiteConnectBadPath(t *testing.T) {
	driver := NewSQLiteDriver()
	err := driver.Connect(ConnectionConfig{})
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestSQLiteGetSchemas(t *testing.T) {
	driver, _ := createTestSQLiteDB(t)
	defer driver.Disconnect()

	schemas, err := driver.GetSchemas("")
	if err != nil {
		t.Fatal(err)
	}
	if len(schemas) != 1 || schemas[0] != "main" {
		t.Errorf("schemas = %v, want [main]", schemas)
	}
}

func TestSQLiteGetTables(t *testing.T) {
	driver, _ := createTestSQLiteDB(t)
	defer driver.Disconnect()

	tables, err := driver.GetTables("", "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 2 {
		t.Fatalf("got %d tables, want 2", len(tables))
	}

	// Check table names
	names := map[string]bool{}
	for _, tbl := range tables {
		names[tbl.Name] = true
	}
	if !names["users"] || !names["posts"] {
		t.Errorf("expected users and posts tables, got %v", names)
	}

	// Check row counts
	for _, tbl := range tables {
		if tbl.Name == "users" && tbl.RowCount != 3 {
			t.Errorf("users row count = %d, want 3", tbl.RowCount)
		}
		if tbl.Name == "posts" && tbl.RowCount != 3 {
			t.Errorf("posts row count = %d, want 3", tbl.RowCount)
		}
	}
}

func TestSQLiteGetColumns(t *testing.T) {
	driver, _ := createTestSQLiteDB(t)
	defer driver.Disconnect()

	cols, err := driver.GetColumns("", "main", "users")
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 4 {
		t.Fatalf("got %d columns, want 4", len(cols))
	}

	// Check id column
	if cols[0].Name != "id" {
		t.Errorf("first column = %q, want id", cols[0].Name)
	}
	if !cols[0].PrimaryKey {
		t.Error("id should be primary key")
	}

	// Check name column
	if cols[1].Name != "name" {
		t.Errorf("second column = %q, want name", cols[1].Name)
	}
	if cols[1].Nullable {
		t.Error("name should not be nullable")
	}
}

func TestSQLiteGetIndexes(t *testing.T) {
	driver, _ := createTestSQLiteDB(t)
	defer driver.Disconnect()

	indexes, err := driver.GetIndexes("", "main", "posts")
	if err != nil {
		t.Fatal(err)
	}
	if len(indexes) == 0 {
		t.Error("expected at least 1 index on posts")
	}

	found := false
	for _, idx := range indexes {
		if idx.Name == "idx_posts_user" {
			found = true
			if len(idx.Columns) != 1 || idx.Columns[0] != "user_id" {
				t.Errorf("index columns = %v, want [user_id]", idx.Columns)
			}
		}
	}
	if !found {
		t.Error("idx_posts_user not found")
	}
}

func TestSQLiteExecuteSelect(t *testing.T) {
	driver, _ := createTestSQLiteDB(t)
	defer driver.Disconnect()

	result, err := driver.Execute("SELECT name, email FROM users ORDER BY name")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Columns) != 2 {
		t.Errorf("columns = %d, want 2", len(result.Columns))
	}
	if result.Columns[0] != "name" || result.Columns[1] != "email" {
		t.Errorf("columns = %v, want [name email]", result.Columns)
	}
	if result.RowCount != 3 {
		t.Errorf("rows = %d, want 3", result.RowCount)
	}
	if result.Rows[0][0] != "Alice" {
		t.Errorf("first row name = %q, want Alice", result.Rows[0][0])
	}
	// Duration may be 0 on very fast machines, just check it's non-negative
	if result.Duration < 0 {
		t.Errorf("duration should be non-negative, got %v", result.Duration)
	}
}

func TestSQLiteExecuteJoin(t *testing.T) {
	driver, _ := createTestSQLiteDB(t)
	defer driver.Disconnect()

	result, err := driver.Execute(`
		SELECT u.name, COUNT(p.id) as post_count
		FROM users u LEFT JOIN posts p ON p.user_id = u.id
		GROUP BY u.name ORDER BY post_count DESC
	`)
	if err != nil {
		t.Fatal(err)
	}
	if result.RowCount != 3 {
		t.Errorf("rows = %d, want 3", result.RowCount)
	}
	// Alice has 2 posts
	if result.Rows[0][0] != "Alice" || result.Rows[0][1] != "2" {
		t.Errorf("first row = %v, want [Alice 2]", result.Rows[0])
	}
}

func TestSQLiteExecuteNullHandling(t *testing.T) {
	driver, _ := createTestSQLiteDB(t)
	defer driver.Disconnect()

	driver.Execute("INSERT INTO users (name, email) VALUES ('NullTest', NULL)")
	result, err := driver.Execute("SELECT email FROM users WHERE name = 'NullTest'")
	if err != nil {
		t.Fatal(err)
	}
	if result.Rows[0][0] != "<NULL>" {
		t.Errorf("null value = %q, want <NULL>", result.Rows[0][0])
	}
}

func TestSQLiteExecuteBadSQL(t *testing.T) {
	driver, _ := createTestSQLiteDB(t)
	defer driver.Disconnect()

	_, err := driver.Execute("SELECT * FROM nonexistent_table")
	if err == nil {
		t.Error("expected error for bad SQL")
	}
}

func TestSQLiteGetTablePreview(t *testing.T) {
	driver, _ := createTestSQLiteDB(t)
	defer driver.Disconnect()

	result, err := driver.GetTablePreview("", "main", "users", 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.RowCount != 2 {
		t.Errorf("preview rows = %d, want 2 (limited)", result.RowCount)
	}
}

func TestSQLiteExplainQuery(t *testing.T) {
	driver, _ := createTestSQLiteDB(t)
	defer driver.Disconnect()

	plan, err := driver.ExplainQuery("SELECT * FROM users WHERE name = 'Alice'")
	if err != nil {
		t.Fatal(err)
	}
	if plan == "" {
		t.Error("explain should return a non-empty plan")
	}
}

func TestSQLiteInsertUpdateDelete(t *testing.T) {
	driver, _ := createTestSQLiteDB(t)
	defer driver.Disconnect()

	// Insert
	_, err := driver.Execute("INSERT INTO users (name, email) VALUES ('Dave', 'dave@test.com')")
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// Verify insert
	result, _ := driver.Execute("SELECT COUNT(*) FROM users")
	if result.Rows[0][0] != "4" {
		t.Errorf("count after insert = %s, want 4", result.Rows[0][0])
	}

	// Update
	_, err = driver.Execute("UPDATE users SET active = 0 WHERE name = 'Dave'")
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	// Verify update
	result, _ = driver.Execute("SELECT active FROM users WHERE name = 'Dave'")
	if result.Rows[0][0] != "0" {
		t.Errorf("active after update = %s, want 0", result.Rows[0][0])
	}

	// Delete
	_, err = driver.Execute("DELETE FROM users WHERE name = 'Dave'")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Verify delete
	result, _ = driver.Execute("SELECT COUNT(*) FROM users")
	if result.Rows[0][0] != "3" {
		t.Errorf("count after delete = %s, want 3", result.Rows[0][0])
	}
}

func TestSQLiteDisconnect(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	driver := NewSQLiteDriver()
	driver.Connect(ConnectionConfig{FilePath: path})

	err := driver.Disconnect()
	if err != nil {
		t.Errorf("disconnect failed: %v", err)
	}

	// Disconnect again should be fine
	err = driver.Disconnect()
	if err != nil {
		t.Errorf("second disconnect should not error: %v", err)
	}
}

func TestSQLiteGetDatabases(t *testing.T) {
	driver, _ := createTestSQLiteDB(t)
	defer driver.Disconnect()

	_, err := driver.GetDatabases()
	if err == nil {
		t.Error("GetDatabases should return error for SQLite (not supported)")
	}
}

func TestSQLiteFileDoesNotExistCreatesIt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "newdb.db")

	driver := NewSQLiteDriver()
	err := driver.Connect(ConnectionConfig{FilePath: path})
	if err != nil {
		t.Fatalf("connecting to new file failed: %v", err)
	}
	defer driver.Disconnect()

	// File should exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}
