package ui

import (
	"testing"
)

func TestDefaultVimBindings(t *testing.T) {
	cfg := DefaultVimBindings()
	if cfg.Mode != KeymodeVim {
		t.Errorf("mode = %q, want vim", cfg.Mode)
	}
	if len(cfg.Bindings) == 0 {
		t.Error("no bindings defined")
	}
}

func TestDefaultEmacsBindings(t *testing.T) {
	cfg := DefaultEmacsBindings()
	if cfg.Mode != KeymodeEmacs {
		t.Errorf("mode = %q, want emacs", cfg.Mode)
	}
	if len(cfg.Bindings) == 0 {
		t.Error("no bindings defined")
	}
}

func TestKeyMapActionFor(t *testing.T) {
	km := NewKeyMap(DefaultVimBindings())

	tests := []struct {
		key    string
		action string
	}{
		{"j", "move_down"},
		{"k", "move_up"},
		{"q", "quit"},
		{"?", "help"},
		{":", "command_palette"},
		{"tab", "next_panel"},
		{"/", "search"},
		{"y", "copy_cell"},
		{"Y", "copy_row"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := km.ActionFor(tt.key)
			if got != tt.action {
				t.Errorf("ActionFor(%q) = %q, want %q", tt.key, got, tt.action)
			}
		})
	}
}

func TestKeyMapKeysFor(t *testing.T) {
	km := NewKeyMap(DefaultVimBindings())

	keys := km.KeysFor("move_down")
	if len(keys) == 0 {
		t.Error("move_down should have at least one key")
	}

	found := false
	for _, k := range keys {
		if k == "j" {
			found = true
		}
	}
	if !found {
		t.Errorf("move_down keys should include 'j', got %v", keys)
	}
}

func TestKeyMapUnknownKey(t *testing.T) {
	km := NewKeyMap(DefaultVimBindings())
	action := km.ActionFor("ctrl+shift+alt+f12")
	if action != "" {
		t.Errorf("unknown key should return empty action, got %q", action)
	}
}
