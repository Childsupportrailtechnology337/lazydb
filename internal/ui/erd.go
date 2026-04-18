package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
)

// ERDTable represents a table in the ERD diagram.
type ERDTable struct {
	Name    string
	Columns []db.Column
}

// ERDRelation represents a foreign key relationship between two tables.
type ERDRelation struct {
	FromTable  string
	FromColumn string
	ToTable    string
	ToColumn   string
}

// ERDPanel renders an ASCII Entity Relationship Diagram.
type ERDPanel struct {
	visible   bool
	tables    []ERDTable
	relations []ERDRelation
	width     int
	height    int
	scrollX   int
	scrollY   int
}

// NewERDPanel creates a new ERD panel.
func NewERDPanel() ERDPanel {
	return ERDPanel{}
}

// Show shows the ERD panel with the given tables and relations.
func (e *ERDPanel) Show(tables []ERDTable, relations []ERDRelation) {
	e.visible = true
	e.tables = tables
	e.relations = relations
	e.scrollX = 0
	e.scrollY = 0
}

// Hide hides the ERD panel.
func (e *ERDPanel) Hide() {
	e.visible = false
}

// IsVisible returns whether the ERD panel is visible.
func (e *ERDPanel) IsVisible() bool {
	return e.visible
}

// SetSize sets the panel dimensions.
func (e *ERDPanel) SetSize(w, h int) {
	e.width = w
	e.height = h
}

// Update handles input for the ERD panel.
func (e *ERDPanel) Update(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		e.scrollY++
	case "k", "up":
		if e.scrollY > 0 {
			e.scrollY--
		}
	case "h", "left":
		if e.scrollX > 0 {
			e.scrollX--
		}
	case "l", "right":
		e.scrollX++
	case "esc", "q":
		e.Hide()
	}
	return nil
}

// View renders the ERD panel.
func (e *ERDPanel) View() string {
	if !e.visible {
		return ""
	}

	diagram := e.renderDiagram()

	title := lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render("  Entity Relationship Diagram")
	footer := MutedText.Render("  j/k scroll  h/l pan  Esc close")

	content := title + "\n\n" + diagram + "\n\n" + footer

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Width(e.width - 4).
		Height(e.height - 4).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(e.width, e.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

// tableBoxWidth calculates the width needed for a table box.
func (e *ERDPanel) tableBoxWidth(t ERDTable) int {
	w := len(t.Name) + 2
	for _, col := range t.Columns {
		label := e.columnLabel(col)
		if len(label)+2 > w {
			w = len(label) + 2
		}
	}
	if w < 14 {
		w = 14
	}
	return w
}

// columnLabel formats a column for display inside a table box.
func (e *ERDPanel) columnLabel(col db.Column) string {
	prefix := " "
	if col.PrimaryKey {
		prefix = "*"
	}
	dtype := col.DataType
	if len(dtype) > 10 {
		dtype = dtype[:10]
	}
	return fmt.Sprintf("%s%-10s %s", prefix, col.Name, dtype)
}

// renderTableBox renders a single table as an ASCII box.
func (e *ERDPanel) renderTableBox(t ERDTable) []string {
	w := e.tableBoxWidth(t)
	var lines []string

	pkStyle := lipgloss.NewStyle().Foreground(AccentColor).Bold(true)
	nameStyle := lipgloss.NewStyle().Foreground(SecondaryColor).Bold(true)

	// Top border
	lines = append(lines, "┌"+strings.Repeat("─", w)+"┐")

	// Table name
	name := t.Name
	padded := name + strings.Repeat(" ", w-len(name))
	lines = append(lines, "│"+nameStyle.Render(padded)+"│")

	// Separator
	lines = append(lines, "├"+strings.Repeat("─", w)+"┤")

	// Columns
	for _, col := range t.Columns {
		label := e.columnLabel(col)
		padLen := w - len(label)
		if padLen < 0 {
			padLen = 0
			label = label[:w]
		}
		cell := label + strings.Repeat(" ", padLen)
		if col.PrimaryKey {
			cell = pkStyle.Render(cell)
		}
		lines = append(lines, "│"+cell+"│")
	}

	// Bottom border
	lines = append(lines, "└"+strings.Repeat("─", w)+"┘")

	return lines
}

// renderDiagram renders the complete ERD diagram.
func (e *ERDPanel) renderDiagram() string {
	if len(e.tables) == 0 {
		return MutedText.Render("  No tables to display")
	}

	// Arrange tables in a grid: determine columns per row based on available width
	colsPerRow := e.gridCols()
	if colsPerRow < 1 {
		colsPerRow = 1
	}

	// Render each table box
	type renderedTable struct {
		lines []string
		width int
	}
	rendered := make([]renderedTable, len(e.tables))
	for i, t := range e.tables {
		lines := e.renderTableBox(t)
		w := e.tableBoxWidth(t) + 2 // +2 for the border chars
		rendered[i] = renderedTable{lines: lines, width: w}
	}

	// Build grid rows
	var outputLines []string
	gap := "     " // horizontal gap between tables

	for rowStart := 0; rowStart < len(rendered); rowStart += colsPerRow {
		rowEnd := rowStart + colsPerRow
		if rowEnd > len(rendered) {
			rowEnd = len(rendered)
		}
		rowTables := rendered[rowStart:rowEnd]

		// Find max height in this row
		maxH := 0
		for _, rt := range rowTables {
			if len(rt.lines) > maxH {
				maxH = len(rt.lines)
			}
		}

		// Render row line by line
		for lineIdx := 0; lineIdx < maxH; lineIdx++ {
			var parts []string
			for _, rt := range rowTables {
				if lineIdx < len(rt.lines) {
					parts = append(parts, rt.lines[lineIdx])
				} else {
					parts = append(parts, strings.Repeat(" ", rt.width))
				}
			}
			outputLines = append(outputLines, strings.Join(parts, gap))
		}

		// Add spacing between grid rows
		outputLines = append(outputLines, "")
	}

	// Append relation annotations
	if len(e.relations) > 0 {
		relStyle := lipgloss.NewStyle().Foreground(AccentColor)
		outputLines = append(outputLines, "")
		for _, rel := range e.relations {
			line := fmt.Sprintf("  FK: %s.%s ─── %s.%s",
				rel.FromTable, rel.FromColumn,
				rel.ToTable, rel.ToColumn)
			outputLines = append(outputLines, relStyle.Render(line))
		}
	}

	// Apply scroll
	if e.scrollY > 0 {
		if e.scrollY < len(outputLines) {
			outputLines = outputLines[e.scrollY:]
		} else {
			outputLines = nil
		}
	}

	// Limit to visible height
	visibleH := e.height - 10
	if visibleH < 1 {
		visibleH = 1
	}
	if len(outputLines) > visibleH {
		outputLines = outputLines[:visibleH]
	}

	// Apply horizontal scroll
	if e.scrollX > 0 {
		for i, line := range outputLines {
			runes := []rune(line)
			if e.scrollX < len(runes) {
				outputLines[i] = string(runes[e.scrollX:])
			} else {
				outputLines[i] = ""
			}
		}
	}

	return strings.Join(outputLines, "\n")
}

// gridCols returns how many table columns fit per row.
func (e *ERDPanel) gridCols() int {
	if len(e.tables) == 0 {
		return 1
	}

	avgWidth := 0
	for _, t := range e.tables {
		avgWidth += e.tableBoxWidth(t) + 2
	}
	avgWidth /= len(e.tables)

	available := e.width - 8
	cols := available / (avgWidth + 5) // 5 for gap
	if cols < 1 {
		cols = 1
	}
	if cols > len(e.tables) {
		cols = len(e.tables)
	}
	return cols
}
