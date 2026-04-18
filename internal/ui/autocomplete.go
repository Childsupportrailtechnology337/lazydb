package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// AutoComplete provides suggestions for the query editor.
type AutoComplete struct {
	suggestions []string
	visible     bool
	cursor      int
	tableNames  []string
	columnNames map[string][]string // table -> columns
}

// NewAutoComplete creates a new auto-complete.
func NewAutoComplete() AutoComplete {
	return AutoComplete{
		columnNames: make(map[string][]string),
	}
}

// SetSchema sets the available table and column names for auto-complete.
func (a *AutoComplete) SetSchema(tables []string, columns map[string][]string) {
	a.tableNames = tables
	a.columnNames = columns
}

// SetSchemaFromNodes extracts table/column names from schema tree nodes.
func (a *AutoComplete) SetSchemaFromNodes(nodes []*TreeNode) {
	a.tableNames = nil
	a.columnNames = make(map[string][]string)
	var walk func([]*TreeNode)
	walk = func(nodes []*TreeNode) {
		for _, n := range nodes {
			switch n.Type {
			case "table":
				a.tableNames = append(a.tableNames, n.Name)
				for _, child := range n.Children {
					if child.Type == "column" {
						a.columnNames[n.Name] = append(a.columnNames[n.Name], child.Name)
					}
				}
			}
			if len(n.Children) > 0 {
				walk(n.Children)
			}
		}
	}
	walk(nodes)
}

var sqlKeywords = []string{
	"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "CREATE",
	"DROP", "ALTER", "TABLE", "INTO", "VALUES", "SET", "JOIN", "LEFT",
	"RIGHT", "INNER", "OUTER", "CROSS", "FULL", "ON", "AND", "OR",
	"NOT", "NULL", "IS", "IN", "LIKE", "ILIKE", "BETWEEN", "ORDER",
	"BY", "GROUP", "HAVING", "LIMIT", "OFFSET", "AS", "DISTINCT",
	"COUNT", "SUM", "AVG", "MAX", "MIN", "DESC", "ASC", "UNION",
	"ALL", "EXISTS", "CASE", "WHEN", "THEN", "ELSE", "END",
	"BEGIN", "COMMIT", "ROLLBACK", "PRIMARY", "KEY", "FOREIGN",
	"REFERENCES", "INDEX", "UNIQUE", "DEFAULT", "CONSTRAINT",
	"CASCADE", "RESTRICT", "TRUNCATE", "EXPLAIN", "ANALYZE",
	"VACUUM", "PRAGMA", "RETURNING", "WITH", "RECURSIVE",
	"COALESCE", "NULLIF", "CAST", "EXTRACT", "INTERVAL",
}

// GetSuggestions returns suggestions for the current word.
func (a *AutoComplete) GetSuggestions(currentWord string) []string {
	if currentWord == "" {
		a.visible = false
		return nil
	}

	prefix := strings.ToLower(currentWord)
	var matches []string

	// Match SQL keywords
	for _, kw := range sqlKeywords {
		if strings.HasPrefix(strings.ToLower(kw), prefix) && strings.ToLower(kw) != prefix {
			matches = append(matches, kw)
		}
	}

	// Match table names
	for _, t := range a.tableNames {
		if strings.HasPrefix(strings.ToLower(t), prefix) {
			matches = append(matches, t)
		}
	}

	// Match column names from all tables
	for _, cols := range a.columnNames {
		for _, c := range cols {
			if strings.HasPrefix(strings.ToLower(c), prefix) {
				matches = append(matches, c)
			}
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, m := range matches {
		lower := strings.ToLower(m)
		if !seen[lower] {
			seen[lower] = true
			unique = append(unique, m)
		}
	}

	if len(unique) > 10 {
		unique = unique[:10]
	}

	a.suggestions = unique
	a.visible = len(unique) > 0
	a.cursor = 0
	return unique
}

// MoveUp moves the selection up.
func (a *AutoComplete) MoveUp() {
	if a.cursor > 0 {
		a.cursor--
	}
}

// MoveDown moves the selection down.
func (a *AutoComplete) MoveDown() {
	if a.cursor < len(a.suggestions)-1 {
		a.cursor++
	}
}

// Accept returns the currently selected suggestion.
func (a *AutoComplete) Accept() string {
	if a.cursor < len(a.suggestions) {
		s := a.suggestions[a.cursor]
		a.Hide()
		return s
	}
	return ""
}

// Hide hides the autocomplete popup.
func (a *AutoComplete) Hide() {
	a.visible = false
	a.suggestions = nil
}

// IsVisible returns whether suggestions are visible.
func (a *AutoComplete) IsVisible() bool {
	return a.visible
}

// View renders the autocomplete popup.
func (a *AutoComplete) View(x, y int) string {
	if !a.visible || len(a.suggestions) == 0 {
		return ""
	}

	var b strings.Builder
	for i, s := range a.suggestions {
		line := "  " + s + "  "
		if i == a.cursor {
			line = SelectedRow.Render(line)
		} else {
			line = lipgloss.NewStyle().Foreground(TextColor).Render(line)
		}
		b.WriteString(line + "\n")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(MutedColor).
		Render(b.String())
}
