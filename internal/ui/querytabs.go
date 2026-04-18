package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// QueryTab represents a single query tab.
type QueryTab struct {
	Name  string
	Query string
}

// QueryTabs manages multiple query editor tabs.
type QueryTabs struct {
	tabs   []QueryTab
	active int
	width  int
}

// NewQueryTabs creates a new query tabs manager.
func NewQueryTabs() QueryTabs {
	return QueryTabs{
		tabs: []QueryTab{
			{Name: "Query 1", Query: ""},
		},
		active: 0,
	}
}

// SetWidth sets the tabs bar width.
func (t *QueryTabs) SetWidth(w int) {
	t.width = w
}

// AddTab adds a new tab and switches to it.
func (t *QueryTabs) AddTab() int {
	idx := len(t.tabs) + 1
	t.tabs = append(t.tabs, QueryTab{
		Name: fmt.Sprintf("Query %d", idx),
	})
	t.active = len(t.tabs) - 1
	return t.active
}

// CloseTab closes the current tab.
func (t *QueryTabs) CloseTab() {
	if len(t.tabs) <= 1 {
		return
	}
	t.tabs = append(t.tabs[:t.active], t.tabs[t.active+1:]...)
	if t.active >= len(t.tabs) {
		t.active = len(t.tabs) - 1
	}
}

// NextTab switches to the next tab.
func (t *QueryTabs) NextTab() {
	t.active = (t.active + 1) % len(t.tabs)
}

// PrevTab switches to the previous tab.
func (t *QueryTabs) PrevTab() {
	t.active = (t.active - 1 + len(t.tabs)) % len(t.tabs)
}

// ActiveTab returns the active tab index.
func (t *QueryTabs) ActiveTab() int {
	return t.active
}

// GetActiveQuery returns the query of the active tab.
func (t *QueryTabs) GetActiveQuery() string {
	return t.tabs[t.active].Query
}

// SetActiveQuery sets the query of the active tab.
func (t *QueryTabs) SetActiveQuery(q string) {
	t.tabs[t.active].Query = q
}

// Count returns the number of tabs.
func (t *QueryTabs) Count() int {
	return len(t.tabs)
}

// View renders the tab bar.
func (t *QueryTabs) View() string {
	var parts []string

	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(PrimaryColor).
		Padding(0, 1)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(MutedColor).
		Padding(0, 1)

	for i, tab := range t.tabs {
		name := tab.Name
		if i == t.active {
			parts = append(parts, activeStyle.Render(name))
		} else {
			parts = append(parts, inactiveStyle.Render(name))
		}
	}

	addBtn := MutedText.Render(" [+]")
	return strings.Join(parts, "") + addBtn
}
