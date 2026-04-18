package query

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// HistoryEntry represents a single query in history.
type HistoryEntry struct {
	Query     string    `json:"query"`
	Database  string    `json:"database"`
	Timestamp time.Time `json:"timestamp"`
	Duration  string    `json:"duration"`
	RowCount  int       `json:"row_count"`
}

// History manages query history.
type History struct {
	Entries []HistoryEntry `json:"entries"`
	MaxSize int            `json:"-"`
	path    string
}

// NewHistory creates a new history manager.
func NewHistory() *History {
	home, _ := os.UserHomeDir()
	return &History{
		MaxSize: 500,
		path:    filepath.Join(home, ".config", "lazydb", "history.json"),
	}
}

// Add adds an entry to history.
func (h *History) Add(entry HistoryEntry) {
	h.Entries = append(h.Entries, entry)
	if len(h.Entries) > h.MaxSize {
		h.Entries = h.Entries[len(h.Entries)-h.MaxSize:]
	}
}

// Load reads history from disk.
func (h *History) Load() error {
	data, err := os.ReadFile(h.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &h.Entries)
}

// Save writes history to disk.
func (h *History) Save() error {
	dir := filepath.Dir(h.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(h.Entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.path, data, 0o644)
}
