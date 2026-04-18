package ui

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

// Theme defines a color theme for the application.
type Theme struct {
	Name       string `json:"name"`
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
	Accent     string `json:"accent"`
	Success    string `json:"success"`
	Error      string `json:"error"`
	Muted      string `json:"muted"`
	Text       string `json:"text"`
	Background string `json:"background"`
	PanelBg    string `json:"panel_bg"`
	Selection  string `json:"selection"`
}

// BuiltinThemes contains all built-in color themes.
var BuiltinThemes = map[string]Theme{
	"default": {
		Name:       "default",
		Primary:    "#7C3AED",
		Secondary:  "#06B6D4",
		Accent:     "#F59E0B",
		Success:    "#10B981",
		Error:      "#EF4444",
		Muted:      "#6B7280",
		Text:       "#E5E7EB",
		Background: "#1F2937",
		PanelBg:    "#111827",
		Selection:  "#7C3AED",
	},
	"catppuccin-mocha": {
		Name:       "catppuccin-mocha",
		Primary:    "#CBA6F7",
		Secondary:  "#89DCEB",
		Accent:     "#F9E2AF",
		Success:    "#A6E3A1",
		Error:      "#F38BA8",
		Muted:      "#6C7086",
		Text:       "#CDD6F4",
		Background: "#1E1E2E",
		PanelBg:    "#181825",
		Selection:  "#585B70",
	},
	"dracula": {
		Name:       "dracula",
		Primary:    "#BD93F9",
		Secondary:  "#8BE9FD",
		Accent:     "#F1FA8C",
		Success:    "#50FA7B",
		Error:      "#FF5555",
		Muted:      "#6272A4",
		Text:       "#F8F8F2",
		Background: "#282A36",
		PanelBg:    "#21222C",
		Selection:  "#44475A",
	},
	"tokyo-night": {
		Name:       "tokyo-night",
		Primary:    "#7AA2F7",
		Secondary:  "#7DCFFF",
		Accent:     "#E0AF68",
		Success:    "#9ECE6A",
		Error:      "#F7768E",
		Muted:      "#565F89",
		Text:       "#C0CAF5",
		Background: "#1A1B26",
		PanelBg:    "#16161E",
		Selection:  "#33467C",
	},
	"gruvbox": {
		Name:       "gruvbox",
		Primary:    "#D3869B",
		Secondary:  "#83A598",
		Accent:     "#FABD2F",
		Success:    "#B8BB26",
		Error:      "#FB4934",
		Muted:      "#928374",
		Text:       "#EBDBB2",
		Background: "#282828",
		PanelBg:    "#1D2021",
		Selection:  "#504945",
	},
	"nord": {
		Name:       "nord",
		Primary:    "#88C0D0",
		Secondary:  "#81A1C1",
		Accent:     "#EBCB8B",
		Success:    "#A3BE8C",
		Error:      "#BF616A",
		Muted:      "#4C566A",
		Text:       "#ECEFF4",
		Background: "#2E3440",
		PanelBg:    "#272C36",
		Selection:  "#434C5E",
	},
	"solarized-dark": {
		Name:       "solarized-dark",
		Primary:    "#268BD2",
		Secondary:  "#2AA198",
		Accent:     "#B58900",
		Success:    "#859900",
		Error:      "#DC322F",
		Muted:      "#586E75",
		Text:       "#839496",
		Background: "#002B36",
		PanelBg:    "#073642",
		Selection:  "#073642",
	},
	"one-dark": {
		Name:       "one-dark",
		Primary:    "#C678DD",
		Secondary:  "#56B6C2",
		Accent:     "#E5C07B",
		Success:    "#98C379",
		Error:      "#E06C75",
		Muted:      "#5C6370",
		Text:       "#ABB2BF",
		Background: "#282C34",
		PanelBg:    "#21252B",
		Selection:  "#3E4451",
	},
	"rose-pine": {
		Name:       "rose-pine",
		Primary:    "#C4A7E7",
		Secondary:  "#9CCFD8",
		Accent:     "#F6C177",
		Success:    "#31748F",
		Error:      "#EB6F92",
		Muted:      "#6E6A86",
		Text:       "#E0DEF4",
		Background: "#191724",
		PanelBg:    "#1F1D2E",
		Selection:  "#26233A",
	},
}

// ApplyTheme applies a theme to the global styles.
func ApplyTheme(t Theme) {
	PrimaryColor = lipgloss.Color(t.Primary)
	SecondaryColor = lipgloss.Color(t.Secondary)
	AccentColor = lipgloss.Color(t.Accent)
	SuccessColor = lipgloss.Color(t.Success)
	ErrorColor = lipgloss.Color(t.Error)
	MutedColor = lipgloss.Color(t.Muted)
	TextColor = lipgloss.Color(t.Text)
	BgColor = lipgloss.Color(t.Background)
	PanelBgColor = lipgloss.Color(t.PanelBg)

	// Rebuild styles with new colors
	ActivePanelBorder = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor)

	InactivePanelBorder = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(MutedColor)

	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(PrimaryColor).
		Padding(0, 1)

	PanelTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(SecondaryColor)

	NormalText = lipgloss.NewStyle().Foreground(TextColor)
	MutedText = lipgloss.NewStyle().Foreground(MutedColor)

	SelectedRow = lipgloss.NewStyle().
		Background(lipgloss.Color(t.Selection)).
		Foreground(lipgloss.Color("#FFFFFF"))

	StatusBarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(t.Background)).
		Foreground(TextColor).
		Padding(0, 1)

	StatusBarKeyStyle = lipgloss.NewStyle().
		Background(PrimaryColor).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1)

	TableHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(SecondaryColor).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(MutedColor)

	NullStyle = lipgloss.NewStyle().
		Foreground(MutedColor).
		Italic(true)

	TreeNodeStyle = lipgloss.NewStyle().Foreground(TextColor)
	TreeNodeSelectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(PrimaryColor)

	TreeBranchStyle = lipgloss.NewStyle().Foreground(MutedColor)

	EditorLineNumber = lipgloss.NewStyle().
		Foreground(MutedColor).
		Width(4).
		Align(lipgloss.Right)

	EditorCursor = lipgloss.NewStyle().
		Background(TextColor).
		Foreground(PanelBgColor)

	HelpKeyStyle = lipgloss.NewStyle().
		Foreground(AccentColor).
		Bold(true)

	HelpDescStyle = lipgloss.NewStyle().Foreground(MutedColor)
}

// LoadThemeFromFile loads a custom theme from a JSON file.
func LoadThemeFromFile(path string) (Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, err
	}
	var t Theme
	if err := json.Unmarshal(data, &t); err != nil {
		return Theme{}, err
	}
	return t, nil
}

// GetTheme returns a theme by name. Falls back to default.
func GetTheme(name string) Theme {
	if t, ok := BuiltinThemes[name]; ok {
		return t
	}
	// Try loading from config dir
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config", "lazydb", "themes", name+".json")
	if t, err := LoadThemeFromFile(path); err == nil {
		return t
	}
	return BuiltinThemes["default"]
}
