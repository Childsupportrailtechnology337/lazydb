package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
)

// diffRowKind indicates how a row differs between A and B.
type diffRowKind int

const (
	diffUnchanged diffRowKind = iota
	diffAdded                 // row only in B
	diffRemoved               // row only in A
	diffChanged               // row exists in both but values differ
)

// diffRow holds a single row for display in the diff viewer.
type diffRow struct {
	kind   diffRowKind
	rowA   []string // nil if added
	rowB   []string // nil if removed
}

// DiffPanel displays two query results side by side with differences highlighted.
type DiffPanel struct {
	visible   bool
	resultA   *db.QueryResult
	resultB   *db.QueryResult
	rows      []diffRow
	cursor    int
	scrollY   int
	width     int
	height    int
	colWidths []int
}

// NewDiffPanel creates a new query diff panel.
func NewDiffPanel() DiffPanel {
	return DiffPanel{}
}

// Show shows the diff panel with two query results.
func (d *DiffPanel) Show(resultA, resultB *db.QueryResult) {
	d.visible = true
	d.resultA = resultA
	d.resultB = resultB
	d.cursor = 0
	d.scrollY = 0
	d.computeDiff()
	d.computeColWidths()
}

// Hide hides the diff panel.
func (d *DiffPanel) Hide() {
	d.visible = false
}

// IsVisible returns whether the diff panel is visible.
func (d *DiffPanel) IsVisible() bool {
	return d.visible
}

// SetSize sets the panel dimensions.
func (d *DiffPanel) SetSize(w, h int) {
	d.width = w
	d.height = h
}

// Update handles input for the diff panel.
func (d *DiffPanel) Update(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		if d.cursor < len(d.rows)-1 {
			d.cursor++
		}
	case "k", "up":
		if d.cursor > 0 {
			d.cursor--
		}
	case "g":
		d.cursor = 0
	case "G":
		if len(d.rows) > 0 {
			d.cursor = len(d.rows) - 1
		}
	case "esc", "q":
		d.Hide()
	}

	// Adjust scroll
	visibleRows := d.visibleRowCount()
	if d.cursor < d.scrollY {
		d.scrollY = d.cursor
	}
	if d.cursor >= d.scrollY+visibleRows {
		d.scrollY = d.cursor - visibleRows + 1
	}

	return nil
}

// View renders the diff panel.
func (d *DiffPanel) View() string {
	if !d.visible {
		return ""
	}

	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render("  Query Diff")
	b.WriteString(title + "\n")

	if d.resultA == nil || d.resultB == nil {
		b.WriteString(MutedText.Render("  Need two results to compare"))
		return d.wrapModal(b.String())
	}

	// Summary
	addedStyle := lipgloss.NewStyle().Foreground(SuccessColor)
	removedStyle := lipgloss.NewStyle().Foreground(ErrorColor)
	changedStyle := lipgloss.NewStyle().Foreground(AccentColor)

	added, removed, changed := d.countChanges()
	summary := fmt.Sprintf("  %s  %s  %s  %s",
		addedStyle.Render(fmt.Sprintf("+%d added", added)),
		removedStyle.Render(fmt.Sprintf("-%d removed", removed)),
		changedStyle.Render(fmt.Sprintf("~%d changed", changed)),
		MutedText.Render(fmt.Sprintf("%d total", len(d.rows))),
	)
	b.WriteString(summary + "\n\n")

	// Column headers (side by side: A | B)
	halfWidth := (d.width - 10) / 2
	headerA := d.renderHeader("A", halfWidth)
	headerB := d.renderHeader("B", halfWidth)
	b.WriteString(headerA + " │ " + headerB + "\n")
	b.WriteString(strings.Repeat("─", halfWidth) + "─┼─" + strings.Repeat("─", halfWidth) + "\n")

	// Rows
	visibleRows := d.visibleRowCount()
	end := d.scrollY + visibleRows
	if end > len(d.rows) {
		end = len(d.rows)
	}

	for i := d.scrollY; i < end; i++ {
		row := d.rows[i]
		lineA := d.renderRowCells(row.rowA, halfWidth)
		lineB := d.renderRowCells(row.rowB, halfWidth)

		var styledA, styledB string
		switch row.kind {
		case diffAdded:
			styledA = MutedText.Render(padRight(strings.Repeat("·", 3), halfWidth))
			styledB = addedStyle.Render(lineB)
		case diffRemoved:
			styledA = removedStyle.Render(lineA)
			styledB = MutedText.Render(padRight(strings.Repeat("·", 3), halfWidth))
		case diffChanged:
			styledA = changedStyle.Render(lineA)
			styledB = changedStyle.Render(lineB)
		default:
			styledA = lineA
			styledB = lineB
		}

		line := styledA + " │ " + styledB
		if i == d.cursor {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("#374151")).
				Render(line)
		}
		b.WriteString(line + "\n")
	}

	// Footer
	b.WriteString("\n")
	if len(d.rows) > 0 {
		b.WriteString(MutedText.Render(fmt.Sprintf("  Row %d/%d", d.cursor+1, len(d.rows))))
	}
	b.WriteString("  " + MutedText.Render("j/k scroll  Esc close"))

	return d.wrapModal(b.String())
}

func (d *DiffPanel) wrapModal(content string) string {
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Width(d.width - 4).
		Height(d.height - 4).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(d.width, d.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

func (d *DiffPanel) visibleRowCount() int {
	v := d.height - 14
	if v < 1 {
		v = 1
	}
	return v
}

func (d *DiffPanel) renderHeader(label string, maxWidth int) string {
	if d.resultA == nil {
		return padRight(label, maxWidth)
	}
	cols := d.resultA.Columns
	if label == "B" && d.resultB != nil {
		cols = d.resultB.Columns
	}
	var parts []string
	used := 0
	for i, col := range cols {
		w := d.colWidth(i)
		if used+w > maxWidth && len(parts) > 0 {
			break
		}
		parts = append(parts, padRight(col, w))
		used += w + 1
	}
	header := strings.Join(parts, " ")
	return TableHeaderStyle.Render(padRight(header, maxWidth))
}

func (d *DiffPanel) renderRowCells(row []string, maxWidth int) string {
	if row == nil {
		return padRight("", maxWidth)
	}
	var parts []string
	used := 0
	for i, cell := range row {
		w := d.colWidth(i)
		if used+w > maxWidth && len(parts) > 0 {
			break
		}
		if len(cell) > w-1 {
			cell = cell[:w-2] + "…"
		}
		parts = append(parts, padRight(cell, w))
		used += w + 1
	}
	line := strings.Join(parts, " ")
	return padRight(line, maxWidth)
}

func (d *DiffPanel) colWidth(i int) int {
	if i < len(d.colWidths) {
		return d.colWidths[i]
	}
	return 10
}

func (d *DiffPanel) computeColWidths() {
	var cols []string
	if d.resultA != nil {
		cols = d.resultA.Columns
	} else if d.resultB != nil {
		cols = d.resultB.Columns
	}
	if len(cols) == 0 {
		d.colWidths = nil
		return
	}

	d.colWidths = make([]int, len(cols))
	for i, col := range cols {
		d.colWidths[i] = len(col) + 2
	}

	// Scan all rows for wider values
	allRows := make([][]string, 0)
	if d.resultA != nil {
		allRows = append(allRows, d.resultA.Rows...)
	}
	if d.resultB != nil {
		allRows = append(allRows, d.resultB.Rows...)
	}
	for _, row := range allRows {
		for i, cell := range row {
			if i < len(d.colWidths) {
				w := len(cell) + 2
				if w > 20 {
					w = 20
				}
				if w > d.colWidths[i] {
					d.colWidths[i] = w
				}
			}
		}
	}
}

// computeDiff compares resultA and resultB row by row.
func (d *DiffPanel) computeDiff() {
	d.rows = nil

	if d.resultA == nil && d.resultB == nil {
		return
	}
	if d.resultA == nil {
		for _, row := range d.resultB.Rows {
			d.rows = append(d.rows, diffRow{kind: diffAdded, rowB: row})
		}
		return
	}
	if d.resultB == nil {
		for _, row := range d.resultA.Rows {
			d.rows = append(d.rows, diffRow{kind: diffRemoved, rowA: row})
		}
		return
	}

	// Build a map of row keys for B for quick lookup
	bUsed := make([]bool, len(d.resultB.Rows))
	bMap := make(map[string][]int) // rowKey -> indices in B
	for i, row := range d.resultB.Rows {
		key := rowKey(row)
		bMap[key] = append(bMap[key], i)
	}

	// Match rows from A
	for _, rowA := range d.resultA.Rows {
		key := rowKey(rowA)
		if indices, ok := bMap[key]; ok && len(indices) > 0 {
			// Exact match found
			idx := indices[0]
			bMap[key] = indices[1:]
			bUsed[idx] = true
			d.rows = append(d.rows, diffRow{
				kind: diffUnchanged,
				rowA: rowA,
				rowB: d.resultB.Rows[idx],
			})
		} else {
			// Try to find a partial match (same number of columns, at least one differs)
			matchIdx := -1
			for j, rowB := range d.resultB.Rows {
				if !bUsed[j] && len(rowA) == len(rowB) && partialMatch(rowA, rowB) {
					matchIdx = j
					break
				}
			}
			if matchIdx >= 0 {
				bUsed[matchIdx] = true
				d.rows = append(d.rows, diffRow{
					kind: diffChanged,
					rowA: rowA,
					rowB: d.resultB.Rows[matchIdx],
				})
			} else {
				d.rows = append(d.rows, diffRow{
					kind: diffRemoved,
					rowA: rowA,
				})
			}
		}
	}

	// Remaining B rows are additions
	for j, rowB := range d.resultB.Rows {
		if !bUsed[j] {
			d.rows = append(d.rows, diffRow{
				kind: diffAdded,
				rowB: rowB,
			})
		}
	}
}

func (d *DiffPanel) countChanges() (added, removed, changed int) {
	for _, r := range d.rows {
		switch r.kind {
		case diffAdded:
			added++
		case diffRemoved:
			removed++
		case diffChanged:
			changed++
		}
	}
	return
}

// rowKey creates a string key from a row for comparison.
func rowKey(row []string) string {
	return strings.Join(row, "\x00")
}

// partialMatch returns true if at least half the columns match (indicating a changed row
// rather than a completely different row).
func partialMatch(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	matches := 0
	for i := range a {
		if a[i] == b[i] {
			matches++
		}
	}
	return matches > 0 && matches >= len(a)/2
}
