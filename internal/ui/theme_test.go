package ui

import (
	"testing"
)

func TestGetThemeBuiltin(t *testing.T) {
	names := []string{"default", "catppuccin-mocha", "dracula", "tokyo-night", "gruvbox", "nord", "solarized-dark", "one-dark", "rose-pine"}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			theme := GetTheme(name)
			if theme.Name != name {
				t.Errorf("theme name = %q, want %q", theme.Name, name)
			}
			if theme.Primary == "" {
				t.Error("primary color is empty")
			}
			if theme.Secondary == "" {
				t.Error("secondary color is empty")
			}
			if theme.Text == "" {
				t.Error("text color is empty")
			}
			if theme.Background == "" {
				t.Error("background color is empty")
			}
		})
	}
}

func TestGetThemeUnknownFallsBack(t *testing.T) {
	theme := GetTheme("nonexistent-theme")
	if theme.Name != "default" {
		t.Errorf("unknown theme should fall back to default, got %q", theme.Name)
	}
}

func TestApplyThemeDoesNotPanic(t *testing.T) {
	for name, theme := range BuiltinThemes {
		t.Run(name, func(t *testing.T) {
			ApplyTheme(theme)
		})
	}
	// Restore default
	ApplyTheme(BuiltinThemes["default"])
}

func TestBuiltinThemeCount(t *testing.T) {
	if len(BuiltinThemes) != 9 {
		t.Errorf("expected 9 builtin themes, got %d", len(BuiltinThemes))
	}
}
