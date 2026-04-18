package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
)

// InlineEditMsg is sent when inline editing produces an UPDATE statement.
type InlineEditMsg struct {
	Table      string
	Column     string
	OldValue   string
	NewValue   string
	PrimaryKey map[string]string // PK column -> value for WHERE clause
	SQL        string
}

// InlineEditor handles inline cell editing in the results panel.
type InlineEditor struct {
	active    bool
	value     string
	cursorX   int
	table     string
	column    string
	rowIdx    int
	colIdx    int
	result    *db.QueryResult
	pkColumns map[string]int // PK column name -> column index
	width     int
}

// NewInlineEditor creates a new inline editor.
func NewInlineEditor() InlineEditor {
	return InlineEditor{}
}

// StartEdit begins inline editing of a cell.
func (e *InlineEditor) StartEdit(result *db.QueryResult, table string, rowIdx, colIdx int, pkCols map[string]int) {
	if rowIdx >= len(result.Rows) || colIdx >= len(result.Columns) {
		return
	}
	e.active = true
	e.result = result
	e.table = table
	e.column = result.Columns[colIdx]
	e.rowIdx = rowIdx
	e.colIdx = colIdx
	e.pkColumns = pkCols
	e.value = result.Rows[rowIdx][colIdx]
	if e.value == "<NULL>" {
		e.value = ""
	}
	e.cursorX = len(e.value)
}

// Cancel cancels editing.
func (e *InlineEditor) Cancel() {
	e.active = false
}

// IsActive returns whether the editor is active.
func (e *InlineEditor) IsActive() bool {
	return e.active
}

// SetWidth sets the editor width.
func (e *InlineEditor) SetWidth(w int) {
	e.width = w
}

// Update handles input for the inline editor.
func (e *InlineEditor) Update(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEsc:
		e.Cancel()
		return nil

	case tea.KeyEnter:
		// Build UPDATE statement
		oldValue := e.result.Rows[e.rowIdx][e.colIdx]
		newValue := e.value

		if oldValue == newValue {
			e.Cancel()
			return nil
		}

		// Build WHERE clause from PK
		pk := make(map[string]string)
		var whereParts []string
		for colName, colIdx := range e.pkColumns {
			val := e.result.Rows[e.rowIdx][colIdx]
			pk[colName] = val
			if val == "<NULL>" {
				whereParts = append(whereParts, fmt.Sprintf("%s IS NULL", colName))
			} else {
				whereParts = append(whereParts, fmt.Sprintf("%s = '%s'", colName, strings.ReplaceAll(val, "'", "''")))
			}
		}

		if len(whereParts) == 0 {
			e.Cancel()
			return nil
		}

		var setVal string
		if newValue == "" {
			setVal = "NULL"
		} else {
			setVal = fmt.Sprintf("'%s'", strings.ReplaceAll(newValue, "'", "''"))
		}

		sql := fmt.Sprintf("UPDATE %s SET %s = %s WHERE %s",
			e.table, e.column, setVal, strings.Join(whereParts, " AND "))

		editMsg := InlineEditMsg{
			Table:      e.table,
			Column:     e.column,
			OldValue:   oldValue,
			NewValue:   newValue,
			PrimaryKey: pk,
			SQL:        sql,
		}

		e.Cancel()
		return func() tea.Msg { return editMsg }

	case tea.KeyLeft:
		if e.cursorX > 0 {
			e.cursorX--
		}
	case tea.KeyRight:
		if e.cursorX < len(e.value) {
			e.cursorX++
		}
	case tea.KeyHome:
		e.cursorX = 0
	case tea.KeyEnd:
		e.cursorX = len(e.value)
	case tea.KeyBackspace:
		if e.cursorX > 0 {
			e.value = e.value[:e.cursorX-1] + e.value[e.cursorX:]
			e.cursorX--
		}
	case tea.KeyDelete:
		if e.cursorX < len(e.value) {
			e.value = e.value[:e.cursorX] + e.value[e.cursorX+1:]
		}
	case tea.KeyRunes:
		ch := string(msg.Runes)
		e.value = e.value[:e.cursorX] + ch + e.value[e.cursorX:]
		e.cursorX += len(ch)
	}
	return nil
}

// View renders the inline editor overlay for the current cell.
func (e *InlineEditor) View() string {
	if !e.active {
		return ""
	}

	label := lipgloss.NewStyle().Bold(true).Foreground(SecondaryColor).
		Render(fmt.Sprintf("Editing %s.%s", e.table, e.column))

	var valView string
	if e.cursorX >= len(e.value) {
		valView = e.value + EditorCursor.Render(" ")
	} else {
		valView = e.value[:e.cursorX] +
			EditorCursor.Render(string(e.value[e.cursorX])) +
			e.value[e.cursorX+1:]
	}

	input := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Padding(0, 1).
		Width(40).
		Render(valView)

	hint := MutedText.Render("Enter to save, Esc to cancel")

	return label + "\n" + input + "\n" + hint
}
