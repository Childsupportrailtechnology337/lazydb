package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommandAction represents an action from the command palette.
type CommandAction struct {
	Name        string
	Description string
	Key         string
	Action      string // Internal action identifier
}

// CommandPaletteMsg is sent when a command is selected.
type CommandPaletteMsg struct {
	Action string
}

// CommandPalette provides a searchable command palette overlay.
type CommandPalette struct {
	visible  bool
	input    string
	cursor   int
	commands []CommandAction
	filtered []CommandAction
	width    int
	height   int
}

// NewCommandPalette creates a new command palette.
func NewCommandPalette() CommandPalette {
	cmds := []CommandAction{
		{Name: "Execute Query", Description: "Run the current query", Key: "Ctrl+Enter", Action: "execute_query"},
		{Name: "Export to CSV", Description: "Export results as CSV", Key: "", Action: "export_csv"},
		{Name: "Export to JSON", Description: "Export results as JSON", Key: "", Action: "export_json"},
		{Name: "Export to SQL", Description: "Export results as SQL INSERT", Key: "", Action: "export_sql"},
		{Name: "Copy Row", Description: "Copy selected row to clipboard", Key: "Y", Action: "copy_row"},
		{Name: "Copy Cell", Description: "Copy selected cell value", Key: "y", Action: "copy_cell"},
		{Name: "Row Detail", Description: "View row as key-value pairs", Key: "Enter", Action: "row_detail"},
		{Name: "Describe Table", Description: "Show table structure", Key: "d", Action: "describe_table"},
		{Name: "Refresh Schema", Description: "Reload the schema tree", Key: "", Action: "refresh_schema"},
		{Name: "Query History", Description: "Show recent queries", Key: "Ctrl+H", Action: "query_history"},
		{Name: "Explain Query", Description: "Show query execution plan", Key: "Ctrl+E", Action: "explain_query"},
		{Name: "New Query Tab", Description: "Open a new query tab", Key: "", Action: "new_tab"},
		{Name: "Clear Editor", Description: "Clear the query editor", Key: "", Action: "clear_editor"},
		{Name: "Toggle Help", Description: "Show/hide help", Key: "?", Action: "toggle_help"},
		{Name: "Quit", Description: "Exit LazyDB", Key: "q", Action: "quit"},
	}
	return CommandPalette{
		commands: cmds,
		filtered: cmds,
	}
}

// Toggle toggles the command palette visibility.
func (c *CommandPalette) Toggle() {
	c.visible = !c.visible
	if c.visible {
		c.input = ""
		c.cursor = 0
		c.filtered = c.commands
	}
}

// IsVisible returns whether the palette is visible.
func (c *CommandPalette) IsVisible() bool {
	return c.visible
}

// SetSize sets the overlay dimensions.
func (c *CommandPalette) SetSize(w, h int) {
	c.width = w
	c.height = h
}

// Update handles input for the command palette.
func (c *CommandPalette) Update(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEsc:
		c.Toggle()
		return nil
	case tea.KeyEnter:
		if c.cursor < len(c.filtered) {
			action := c.filtered[c.cursor].Action
			c.Toggle()
			return func() tea.Msg {
				return CommandPaletteMsg{Action: action}
			}
		}
	case tea.KeyUp:
		if c.cursor > 0 {
			c.cursor--
		}
	case tea.KeyDown:
		if c.cursor < len(c.filtered)-1 {
			c.cursor++
		}
	case tea.KeyBackspace:
		if len(c.input) > 0 {
			c.input = c.input[:len(c.input)-1]
			c.filterCommands()
		}
	case tea.KeyRunes:
		c.input += string(msg.Runes)
		c.filterCommands()
	}
	return nil
}

func (c *CommandPalette) filterCommands() {
	if c.input == "" {
		c.filtered = c.commands
		c.cursor = 0
		return
	}

	query := strings.ToLower(c.input)
	c.filtered = nil
	for _, cmd := range c.commands {
		name := strings.ToLower(cmd.Name)
		desc := strings.ToLower(cmd.Description)
		if strings.Contains(name, query) || strings.Contains(desc, query) {
			c.filtered = append(c.filtered, cmd)
		}
	}
	if c.cursor >= len(c.filtered) {
		c.cursor = 0
	}
}

// View renders the command palette.
func (c *CommandPalette) View() string {
	if !c.visible {
		return ""
	}

	var b strings.Builder

	// Search input
	prompt := lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("> ")
	inputStyle := lipgloss.NewStyle().Foreground(TextColor)
	cursorChar := EditorCursor.Render(" ")
	b.WriteString(prompt + inputStyle.Render(c.input) + cursorChar + "\n")
	b.WriteString(MutedText.Render(strings.Repeat("─", 46)) + "\n")

	// Commands list
	maxVisible := 12
	if len(c.filtered) < maxVisible {
		maxVisible = len(c.filtered)
	}

	start := 0
	if c.cursor >= maxVisible {
		start = c.cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(c.filtered) {
		end = len(c.filtered)
	}

	for i := start; i < end; i++ {
		cmd := c.filtered[i]
		name := cmd.Name
		desc := MutedText.Render(" " + cmd.Description)
		key := ""
		if cmd.Key != "" {
			key = " " + HelpKeyStyle.Render("["+cmd.Key+"]")
		}

		line := name + desc + key
		if i == c.cursor {
			line = SelectedRow.Render(padRight(name, 20)) + desc + key
		}
		b.WriteString(line + "\n")
	}

	if len(c.filtered) == 0 {
		b.WriteString(MutedText.Render("  No matching commands") + "\n")
	}

	b.WriteString("\n" + MutedText.Render("  Esc to close"))

	modalW := 50
	if modalW > c.width-4 {
		modalW = c.width - 4
	}

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Width(modalW).
		Padding(1, 2).
		Render(b.String())

	return lipgloss.Place(c.width, c.height,
		lipgloss.Center, lipgloss.Top,
		modal,
		lipgloss.WithWhitespaceChars(" "),
	)
}
