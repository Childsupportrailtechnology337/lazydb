package config

import (
	"os"
	"path/filepath"

	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	Connections []db.ConnectionConfig `yaml:"connections"`
}

// ConfigDir returns the configuration directory path.
func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "lazydb")
}

// Load reads the configuration from disk.
func Load() (*Config, error) {
	dir := ConfigDir()
	path := filepath.Join(dir, "connections.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes the configuration to disk.
func Save(cfg *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "connections.yaml"), data, 0o644)
}
