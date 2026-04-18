package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// KeybindingMode represents a keybinding mode.
type KeybindingMode string

const (
	KeymodeVim   KeybindingMode = "vim"
	KeymodeEmacs KeybindingMode = "emacs"
)

// Keybinding maps an action to a key.
type Keybinding struct {
	Action string `json:"action"`
	Key    string `json:"key"`
	Alt    string `json:"alt,omitempty"` // Alternative key
}

// KeybindingConfig holds all keybinding settings.
type KeybindingConfig struct {
	Mode     KeybindingMode `json:"mode"`
	Bindings []Keybinding   `json:"bindings"`
}

// DefaultVimBindings returns the default vim-style keybindings.
func DefaultVimBindings() KeybindingConfig {
	return KeybindingConfig{
		Mode: KeymodeVim,
		Bindings: []Keybinding{
			{Action: "move_up", Key: "k", Alt: "up"},
			{Action: "move_down", Key: "j", Alt: "down"},
			{Action: "move_left", Key: "h", Alt: "left"},
			{Action: "move_right", Key: "l", Alt: "right"},
			{Action: "go_top", Key: "g"},
			{Action: "go_bottom", Key: "G"},
			{Action: "expand_collapse", Key: "enter"},
			{Action: "preview_table", Key: " "},
			{Action: "execute_query", Key: "ctrl+enter", Alt: "ctrl+j"},
			{Action: "next_panel", Key: "tab"},
			{Action: "prev_panel", Key: "shift+tab"},
			{Action: "panel_schema", Key: "1"},
			{Action: "panel_editor", Key: "2"},
			{Action: "panel_results", Key: "3"},
			{Action: "help", Key: "?"},
			{Action: "command_palette", Key: ":"},
			{Action: "quit", Key: "q"},
			{Action: "force_quit", Key: "ctrl+c"},
			{Action: "search", Key: "/"},
			{Action: "copy_cell", Key: "y"},
			{Action: "copy_row", Key: "Y"},
			{Action: "export", Key: "e"},
			{Action: "row_detail", Key: "enter"},
			{Action: "next_page", Key: "n"},
			{Action: "prev_page", Key: "p"},
			{Action: "new_tab", Key: "ctrl+t"},
			{Action: "close_tab", Key: "ctrl+w"},
			{Action: "explain_query", Key: "ctrl+e"},
			{Action: "history_prev", Key: "alt+up"},
			{Action: "history_next", Key: "alt+down"},
			{Action: "describe_table", Key: "d"},
			{Action: "erd_view", Key: "ctrl+r"},
			{Action: "diff_view", Key: "ctrl+d"},
			{Action: "generate_data", Key: "ctrl+g"},
			{Action: "transaction_begin", Key: "ctrl+b"},
		},
	}
}

// DefaultEmacsBindings returns emacs-style keybindings.
func DefaultEmacsBindings() KeybindingConfig {
	return KeybindingConfig{
		Mode: KeymodeEmacs,
		Bindings: []Keybinding{
			{Action: "move_up", Key: "ctrl+p", Alt: "up"},
			{Action: "move_down", Key: "ctrl+n", Alt: "down"},
			{Action: "move_left", Key: "ctrl+b", Alt: "left"},
			{Action: "move_right", Key: "ctrl+f", Alt: "right"},
			{Action: "go_top", Key: "alt+<"},
			{Action: "go_bottom", Key: "alt+>"},
			{Action: "expand_collapse", Key: "enter"},
			{Action: "preview_table", Key: " "},
			{Action: "execute_query", Key: "ctrl+enter", Alt: "ctrl+j"},
			{Action: "next_panel", Key: "ctrl+o"},
			{Action: "prev_panel", Key: "ctrl+shift+o"},
			{Action: "help", Key: "ctrl+h"},
			{Action: "command_palette", Key: "alt+x"},
			{Action: "quit", Key: "ctrl+q"},
			{Action: "force_quit", Key: "ctrl+c"},
			{Action: "search", Key: "ctrl+s"},
			{Action: "copy_cell", Key: "alt+w"},
			{Action: "copy_row", Key: "ctrl+alt+w"},
			{Action: "export", Key: "ctrl+x ctrl+e"},
			{Action: "row_detail", Key: "enter"},
			{Action: "next_page", Key: "ctrl+v"},
			{Action: "prev_page", Key: "alt+v"},
			{Action: "new_tab", Key: "ctrl+t"},
			{Action: "close_tab", Key: "ctrl+w"},
			{Action: "explain_query", Key: "ctrl+e"},
		},
	}
}

// KeyMap provides quick action-to-key lookup.
type KeyMap struct {
	bindings map[string][]string // action -> keys
	reverse  map[string]string   // key -> action
}

// NewKeyMap creates a key map from a keybinding config.
func NewKeyMap(cfg KeybindingConfig) *KeyMap {
	km := &KeyMap{
		bindings: make(map[string][]string),
		reverse:  make(map[string]string),
	}
	for _, b := range cfg.Bindings {
		km.bindings[b.Action] = append(km.bindings[b.Action], b.Key)
		km.reverse[b.Key] = b.Action
		if b.Alt != "" {
			km.bindings[b.Action] = append(km.bindings[b.Action], b.Alt)
			km.reverse[b.Alt] = b.Action
		}
	}
	return km
}

// ActionFor returns the action for a given key press.
func (km *KeyMap) ActionFor(key string) string {
	return km.reverse[key]
}

// KeysFor returns the keys for a given action.
func (km *KeyMap) KeysFor(action string) []string {
	return km.bindings[action]
}

// LoadKeybindingsFromFile loads keybindings from a JSON file.
func LoadKeybindingsFromFile(path string) (KeybindingConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return KeybindingConfig{}, err
	}
	var cfg KeybindingConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return KeybindingConfig{}, err
	}
	return cfg, nil
}

// LoadKeybindings loads keybindings from the config directory.
func LoadKeybindings() KeybindingConfig {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config", "lazydb", "keybindings.json")
	cfg, err := LoadKeybindingsFromFile(path)
	if err != nil {
		return DefaultVimBindings()
	}
	return cfg
}

// SaveKeybindings saves keybindings to the config directory.
func SaveKeybindings(cfg KeybindingConfig) error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "lazydb")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "keybindings.json"), data, 0o644)
}
