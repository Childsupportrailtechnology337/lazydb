package query

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Bookmark represents a saved query.
type Bookmark struct {
	Name     string `json:"name"`
	Query    string `json:"query"`
	Database string `json:"database"`
}

// Bookmarks manages saved queries.
type Bookmarks struct {
	Items []Bookmark `json:"bookmarks"`
	path  string
}

// NewBookmarks creates a new bookmarks manager.
func NewBookmarks() *Bookmarks {
	home, _ := os.UserHomeDir()
	return &Bookmarks{
		path: filepath.Join(home, ".config", "lazydb", "bookmarks.json"),
	}
}

// Add adds a bookmark.
func (b *Bookmarks) Add(bm Bookmark) {
	b.Items = append(b.Items, bm)
}

// Remove removes a bookmark by index.
func (b *Bookmarks) Remove(idx int) {
	if idx >= 0 && idx < len(b.Items) {
		b.Items = append(b.Items[:idx], b.Items[idx+1:]...)
	}
}

// Load reads bookmarks from disk.
func (b *Bookmarks) Load() error {
	data, err := os.ReadFile(b.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &b.Items)
}

// Save writes bookmarks to disk.
func (b *Bookmarks) Save() error {
	dir := filepath.Dir(b.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(b.Items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(b.path, data, 0o644)
}
