package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
)

// RowDetailPanel shows a single row as key-value pairs.
type RowDetailPanel struct {
	visible  bool
	result   *db.QueryResult
	rowIdx   int
	scrollY  int
	width    int
	height   int
}

// NewRowDetailPanel creates a new row detail panel.
func NewRowDetailPanel() RowDetailPanel {
	return RowDetailPanel{}
}

// Show displays the row detail for the given row index.
func (r *RowDetailPanel) Show(result *db.QueryResult, rowIdx int) {
	r.visible = true
	r.result = result
	r.rowIdx = rowIdx
	r.scrollY = 0
}

// Hide hides the row detail panel.
func (r *RowDetailPanel) Hide() {
	r.visible = false
}

// IsVisible returns whether the panel is visible.
func (r *RowDetailPanel) IsVisible() bool {
	return r.visible
}

// SetSize sets the overlay dimensions.
func (r *RowDetailPanel) SetSize(w, h int) {
	r.width = w
	r.height = h
}

// Update handles input for the row detail panel.
func (r *RowDetailPanel) Update(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "q", "enter":
		r.Hide()
	case "j", "down":
		r.scrollY++
	case "k", "up":
		if r.scrollY > 0 {
			r.scrollY--
		}
	}
	return nil
}

// View renders the row detail panel.
func (r *RowDetailPanel) View() string {
	if !r.visible || r.result == nil || r.rowIdx >= len(r.result.Rows) {
		return ""
	}

	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).
		Render(fmt.Sprintf("  Row %d Detail", r.rowIdx+1))
	b.WriteString(title + "\n\n")

	row := r.result.Rows[r.rowIdx]

	// Find max column name length for alignment
	maxKeyLen := 0
	for _, col := range r.result.Columns {
		if len(col) > maxKeyLen {
			maxKeyLen = len(col)
		}
	}

	modalH := r.height - 8
	if modalH < 10 {
		modalH = 10
	}

	visibleRows := modalH - 6
	end := r.scrollY + visibleRows
	if end > len(r.result.Columns) {
		end = len(r.result.Columns)
	}

	for i := r.scrollY; i < end; i++ {
		col := r.result.Columns[i]
		val := ""
		if i < len(row) {
			val = row[i]
		}

		key := lipgloss.NewStyle().Foreground(SecondaryColor).Bold(true).
			Render(padRight(col, maxKeyLen+2))

		var valStyled string
		if val == "<NULL>" {
			valStyled = NullStyle.Render("NULL")
		} else {
			valStyled = lipgloss.NewStyle().Foreground(TextColor).Render(val)
		}

		b.WriteString("  " + key + valStyled + "\n")
	}

	if end < len(r.result.Columns) {
		b.WriteString(MutedText.Render(fmt.Sprintf("\n  ... %d more fields (j/k to scroll)", len(r.result.Columns)-end)))
	}

	b.WriteString("\n\n" + MutedText.Render("  Press Esc to close"))

	modalW := r.width / 2
	if modalW < 50 {
		modalW = 50
	}
	if modalW > r.width-4 {
		modalW = r.width - 4
	}

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Width(modalW).
		Height(modalH).
		Render(b.String())

	return lipgloss.Place(r.width, r.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}
