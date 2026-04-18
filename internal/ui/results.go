package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
)

// ResultsPanel displays query results in a table.
type ResultsPanel struct {
	result   *db.QueryResult
	cursorR  int // Row cursor
	cursorC  int // Column cursor (for horizontal scroll)
	scrollX  int
	scrollY  int
	width    int
	height   int
	active   bool
	colWidths []int
	page     int
	pageSize int
	message  string // Status message or error
}

// NewResultsPanel creates a new results panel.
func NewResultsPanel() ResultsPanel {
	return ResultsPanel{
		pageSize: 50,
	}
}

// SetSize sets the panel dimensions.
func (r *ResultsPanel) SetSize(w, h int) {
	r.width = w
	r.height = h
}

// SetActive sets whether this panel is focused.
func (r *ResultsPanel) SetActive(active bool) {
	r.active = active
}

// SetResult sets the query result to display.
func (r *ResultsPanel) SetResult(result *db.QueryResult) {
	r.result = result
	r.cursorR = 0
	r.cursorC = 0
	r.scrollX = 0
	r.scrollY = 0
	r.page = 0
	r.message = ""
	r.calculateColWidths()
}

// CursorRow returns the currently selected row index.
func (r *ResultsPanel) CursorRow() int {
	return r.cursorR
}

// SetMessage sets a status message (e.g., error or info).
func (r *ResultsPanel) SetMessage(msg string) {
	r.message = msg
}

func (r *ResultsPanel) calculateColWidths() {
	if r.result == nil {
		r.colWidths = nil
		return
	}

	r.colWidths = make([]int, len(r.result.Columns))
	for i, col := range r.result.Columns {
		r.colWidths[i] = len(col)
	}
	for _, row := range r.result.Rows {
		for i, cell := range row {
			if i < len(r.colWidths) && len(cell) > r.colWidths[i] {
				w := len(cell)
				if w > 40 {
					w = 40 // Cap column width
				}
				r.colWidths[i] = w
			}
		}
	}
	// Add padding
	for i := range r.colWidths {
		r.colWidths[i] += 2
	}
}

// Update handles input for the results panel.
func (r *ResultsPanel) Update(msg tea.KeyMsg) tea.Cmd {
	if r.result == nil {
		return nil
	}

	switch msg.String() {
	case "j", "down":
		if r.cursorR < len(r.result.Rows)-1 {
			r.cursorR++
		}
	case "k", "up":
		if r.cursorR > 0 {
			r.cursorR--
		}
	case "h", "left":
		if r.scrollX > 0 {
			r.scrollX--
		}
	case "l", "right":
		if r.result != nil && r.scrollX < len(r.result.Columns)-1 {
			r.scrollX++
		}
	case "g":
		r.cursorR = 0
	case "G":
		r.cursorR = len(r.result.Rows) - 1
	case "n":
		// Next page
		maxPage := (len(r.result.Rows) - 1) / r.pageSize
		if r.page < maxPage {
			r.page++
			r.cursorR = r.page * r.pageSize
		}
	case "p":
		// Previous page
		if r.page > 0 {
			r.page--
			r.cursorR = r.page * r.pageSize
		}
	}

	// Adjust scroll
	visibleRows := r.height - 6
	if visibleRows < 1 {
		visibleRows = 1
	}
	if r.cursorR < r.scrollY {
		r.scrollY = r.cursorR
	}
	if r.cursorR >= r.scrollY+visibleRows {
		r.scrollY = r.cursorR - visibleRows + 1
	}

	return nil
}

// View renders the results panel.
func (r *ResultsPanel) View() string {
	var b strings.Builder

	title := PanelTitleStyle.Render("Results")

	if r.result != nil {
		info := MutedText.Render(fmt.Sprintf(" (%d rows, %s)", r.result.RowCount, r.result.Duration))
		b.WriteString(title + info + "\n")
	} else {
		b.WriteString(title + "\n")
	}

	if r.message != "" {
		b.WriteString("\n  " + r.message + "\n")
		return r.wrapInBorder(b.String())
	}

	if r.result == nil {
		b.WriteString(MutedText.Render("  Run a query to see results"))
		return r.wrapInBorder(b.String())
	}

	if r.result.Message != "" && len(r.result.Rows) == 0 {
		b.WriteString("\n  " + lipgloss.NewStyle().Foreground(SuccessColor).Render(r.result.Message) + "\n")
		return r.wrapInBorder(b.String())
	}

	if len(r.result.Columns) == 0 {
		b.WriteString(MutedText.Render("  No columns"))
		return r.wrapInBorder(b.String())
	}

	// Determine visible columns based on scrollX and width
	availableWidth := r.width - 4
	visibleCols := r.getVisibleColumns(availableWidth)

	// Render header
	var headerParts []string
	for _, ci := range visibleCols {
		headerParts = append(headerParts, padRight(r.result.Columns[ci], r.colWidths[ci]))
	}
	header := TableHeaderStyle.Render(strings.Join(headerParts, "│"))
	b.WriteString(header + "\n")

	// Separator
	var sepParts []string
	for _, ci := range visibleCols {
		sepParts = append(sepParts, strings.Repeat("─", r.colWidths[ci]))
	}
	b.WriteString(MutedText.Render(strings.Join(sepParts, "┼")) + "\n")

	// Rows
	visibleRows := r.height - 6
	if visibleRows < 1 {
		visibleRows = 1
	}

	end := r.scrollY + visibleRows
	if end > len(r.result.Rows) {
		end = len(r.result.Rows)
	}

	for i := r.scrollY; i < end; i++ {
		row := r.result.Rows[i]
		var cellParts []string
		for _, ci := range visibleCols {
			cell := ""
			if ci < len(row) {
				cell = row[ci]
			}
			if cell == "<NULL>" {
				cell = NullStyle.Render(padRight("NULL", r.colWidths[ci]))
			} else {
				if len(cell) > r.colWidths[ci]-2 {
					cell = cell[:r.colWidths[ci]-3] + "…"
				}
				cell = padRight(cell, r.colWidths[ci])
			}
			cellParts = append(cellParts, cell)
		}
		line := strings.Join(cellParts, "│")

		if i == r.cursorR && r.active {
			line = SelectedRow.Render(line)
		}
		b.WriteString(line + "\n")
	}

	// Page info
	if len(r.result.Rows) > 0 {
		totalPages := (len(r.result.Rows)-1)/r.pageSize + 1
		currentPage := r.cursorR/r.pageSize + 1
		pageInfo := MutedText.Render(fmt.Sprintf("  Row %d/%d  Page %d/%d", r.cursorR+1, len(r.result.Rows), currentPage, totalPages))
		b.WriteString(pageInfo)
	}

	return r.wrapInBorder(b.String())
}

func (r *ResultsPanel) getVisibleColumns(availableWidth int) []int {
	var cols []int
	usedWidth := 0
	for i := r.scrollX; i < len(r.colWidths); i++ {
		if usedWidth+r.colWidths[i] > availableWidth && len(cols) > 0 {
			break
		}
		cols = append(cols, i)
		usedWidth += r.colWidths[i] + 1 // +1 for separator
	}
	return cols
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func (r *ResultsPanel) wrapInBorder(content string) string {
	style := InactivePanelBorder
	if r.active {
		style = ActivePanelBorder
	}
	return style.Width(r.width).Height(r.height).Render(content)
}
