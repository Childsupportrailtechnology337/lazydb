package config

import (
	"os"
	"path/filepath"

	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
	"gopkg.in/yaml.v3"
)

// ProjectConfig represents a per-project .lazydb.yaml configuration.
type ProjectConfig struct {
	Connections []db.ConnectionConfig `yaml:"connections"`
	DefaultConn string               `yaml:"default_connection"`
	Theme       string               `yaml:"theme"`
	Keybindings string               `yaml:"keybindings"` // "vim" or "emacs" or path to file
	StartupSQL  []string             `yaml:"startup_sql"`  // SQL to run on connect
	SavedQueries []SavedQuery        `yaml:"saved_queries"`
	Settings    ProjectSettings      `yaml:"settings"`
}

// SavedQuery represents a saved query in project config.
type SavedQuery struct {
	Name     string `yaml:"name"`
	Query    string `yaml:"query"`
	Database string `yaml:"database"`
}

// ProjectSettings holds project-level settings.
type ProjectSettings struct {
	PageSize    int  `yaml:"page_size"`
	MaxRows     int  `yaml:"max_rows"`
	ConfirmDML  bool `yaml:"confirm_dml"`   // Confirm INSERT/UPDATE/DELETE
	AutoCommit  bool `yaml:"auto_commit"`
	ShowRowNum  bool `yaml:"show_row_numbers"`
	NullDisplay string `yaml:"null_display"` // How to display NULL values
}

// DefaultProjectSettings returns sensible defaults.
func DefaultProjectSettings() ProjectSettings {
	return ProjectSettings{
		PageSize:    50,
		MaxRows:     10000,
		ConfirmDML:  true,
		AutoCommit:  true,
		ShowRowNum:  false,
		NullDisplay: "NULL",
	}
}

// LoadProjectConfig tries to find and load a .lazydb.yaml file.
// It searches the current directory and parent directories.
func LoadProjectConfig() (*ProjectConfig, string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}

	for {
		path := filepath.Join(dir, ".lazydb.yaml")
		data, err := os.ReadFile(path)
		if err == nil {
			var cfg ProjectConfig
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, path, err
			}
			if cfg.Settings.PageSize == 0 {
				cfg.Settings = DefaultProjectSettings()
			}
			return &cfg, path, nil
		}

		// Also try .lazydb.yml
		path = filepath.Join(dir, ".lazydb.yml")
		data, err = os.ReadFile(path)
		if err == nil {
			var cfg ProjectConfig
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, path, err
			}
			if cfg.Settings.PageSize == 0 {
				cfg.Settings = DefaultProjectSettings()
			}
			return &cfg, path, nil
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return nil, "", os.ErrNotExist
}

// SaveProjectConfig writes a project config to the given path.
func SaveProjectConfig(cfg *ProjectConfig, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ExampleProjectConfig returns an example .lazydb.yaml for documentation.
func ExampleProjectConfig() string {
	return `# .lazydb.yaml — LazyDB per-project configuration
# Place this file in your project root.

# Database connections for this project
connections:
  - name: dev-db
    type: postgres
    host: localhost
    port: 5432
    user: postgres
    password: postgres
    database: myapp_dev

  - name: test-db
    type: sqlite
    file_path: ./test.sqlite

# Which connection to use by default
default_connection: dev-db

# Color theme (default, catppuccin-mocha, dracula, tokyo-night, gruvbox, nord, solarized-dark, one-dark, rose-pine)
theme: catppuccin-mocha

# Keybinding mode (vim or emacs)
keybindings: vim

# SQL to run on connect (e.g., SET search_path)
startup_sql:
  - SET search_path TO public, shared;

# Saved queries
saved_queries:
  - name: Active Users
    query: SELECT * FROM users WHERE active = true ORDER BY created_at DESC LIMIT 100

  - name: Order Summary
    query: |
      SELECT u.name, COUNT(o.id) as total_orders, SUM(o.total) as revenue
      FROM users u
      JOIN orders o ON o.user_id = u.id
      GROUP BY u.name
      ORDER BY revenue DESC

# Settings
settings:
  page_size: 50
  max_rows: 10000
  confirm_dml: true
  auto_commit: true
  show_row_numbers: false
  null_display: "NULL"
`
}
