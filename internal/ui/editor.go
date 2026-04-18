package ui

import (
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ExecuteQueryMsg signals that the current query should be executed.
type ExecuteQueryMsg struct {
	Query string
}

// EditorPanel is the query editor panel.
type EditorPanel struct {
	lines    []string
	cursorX  int
	cursorY  int
	width    int
	height   int
	active   bool
	scrollY  int
	history  []string
	histIdx  int
}

// NewEditorPanel creates a new query editor panel.
func NewEditorPanel() EditorPanel {
	return EditorPanel{
		lines:   []string{""},
		histIdx: -1,
	}
}

// SetSize sets the panel dimensions.
func (e *EditorPanel) SetSize(w, h int) {
	e.width = w
	e.height = h
}

// SetActive sets whether this panel is focused.
func (e *EditorPanel) SetActive(active bool) {
	e.active = active
}

// GetQuery returns the current query text.
func (e *EditorPanel) GetQuery() string {
	return strings.Join(e.lines, "\n")
}

// SetQuery sets the editor content.
func (e *EditorPanel) SetQuery(q string) {
	if q == "" {
		e.lines = []string{""}
	} else {
		e.lines = strings.Split(q, "\n")
	}
	e.cursorY = len(e.lines) - 1
	e.cursorX = len(e.lines[e.cursorY])
}

// AddToHistory adds a query to the history.
func (e *EditorPanel) AddToHistory(q string) {
	q = strings.TrimSpace(q)
	if q == "" {
		return
	}
	// Avoid duplicates at the end
	if len(e.history) > 0 && e.history[len(e.history)-1] == q {
		return
	}
	e.history = append(e.history, q)
	e.histIdx = len(e.history)
}

// Update handles input for the editor panel.
func (e *EditorPanel) Update(msg tea.KeyMsg) tea.Cmd {
	switch {
	case msg.String() == "ctrl+enter" || msg.String() == "ctrl+j":
		query := e.GetQuery()
		if strings.TrimSpace(query) != "" {
			return func() tea.Msg {
				return ExecuteQueryMsg{Query: query}
			}
		}
		return nil

	case msg.Type == tea.KeyUp:
		if msg.Alt {
			// Alt+Up: history previous
			if len(e.history) > 0 && e.histIdx > 0 {
				e.histIdx--
				e.SetQuery(e.history[e.histIdx])
			}
			return nil
		}
		if e.cursorY > 0 {
			e.cursorY--
			if e.cursorX > len(e.lines[e.cursorY]) {
				e.cursorX = len(e.lines[e.cursorY])
			}
		}

	case msg.Type == tea.KeyDown:
		if msg.Alt {
			// Alt+Down: history next
			if e.histIdx < len(e.history)-1 {
				e.histIdx++
				e.SetQuery(e.history[e.histIdx])
			}
			return nil
		}
		if e.cursorY < len(e.lines)-1 {
			e.cursorY++
			if e.cursorX > len(e.lines[e.cursorY]) {
				e.cursorX = len(e.lines[e.cursorY])
			}
		}

	case msg.Type == tea.KeyLeft:
		if e.cursorX > 0 {
			e.cursorX--
		} else if e.cursorY > 0 {
			e.cursorY--
			e.cursorX = len(e.lines[e.cursorY])
		}

	case msg.Type == tea.KeyRight:
		if e.cursorX < len(e.lines[e.cursorY]) {
			e.cursorX++
		} else if e.cursorY < len(e.lines)-1 {
			e.cursorY++
			e.cursorX = 0
		}

	case msg.Type == tea.KeyHome:
		e.cursorX = 0

	case msg.Type == tea.KeyEnd:
		e.cursorX = len(e.lines[e.cursorY])

	case msg.Type == tea.KeyBackspace:
		if e.cursorX > 0 {
			line := e.lines[e.cursorY]
			e.lines[e.cursorY] = line[:e.cursorX-1] + line[e.cursorX:]
			e.cursorX--
		} else if e.cursorY > 0 {
			// Merge with previous line
			prev := e.lines[e.cursorY-1]
			e.cursorX = len(prev)
			e.lines[e.cursorY-1] = prev + e.lines[e.cursorY]
			e.lines = append(e.lines[:e.cursorY], e.lines[e.cursorY+1:]...)
			e.cursorY--
		}

	case msg.Type == tea.KeyDelete:
		line := e.lines[e.cursorY]
		if e.cursorX < len(line) {
			e.lines[e.cursorY] = line[:e.cursorX] + line[e.cursorX+1:]
		} else if e.cursorY < len(e.lines)-1 {
			// Merge with next line
			e.lines[e.cursorY] = line + e.lines[e.cursorY+1]
			e.lines = append(e.lines[:e.cursorY+1], e.lines[e.cursorY+2:]...)
		}

	case msg.Type == tea.KeyEnter:
		// Split line at cursor
		line := e.lines[e.cursorY]
		before := line[:e.cursorX]
		after := line[e.cursorX:]
		e.lines[e.cursorY] = before
		newLines := make([]string, len(e.lines)+1)
		copy(newLines, e.lines[:e.cursorY+1])
		newLines[e.cursorY+1] = after
		copy(newLines[e.cursorY+2:], e.lines[e.cursorY+1:])
		e.lines = newLines
		e.cursorY++
		e.cursorX = 0

	case msg.Type == tea.KeyRunes:
		line := e.lines[e.cursorY]
		ch := string(msg.Runes)
		e.lines[e.cursorY] = line[:e.cursorX] + ch + line[e.cursorX:]
		e.cursorX += len(ch)

	case msg.Type == tea.KeyTab:
		// Insert spaces for tab
		line := e.lines[e.cursorY]
		e.lines[e.cursorY] = line[:e.cursorX] + "  " + line[e.cursorX:]
		e.cursorX += 2
	}

	// Adjust scroll
	visibleLines := e.height - 4
	if visibleLines < 1 {
		visibleLines = 1
	}
	if e.cursorY < e.scrollY {
		e.scrollY = e.cursorY
	}
	if e.cursorY >= e.scrollY+visibleLines {
		e.scrollY = e.cursorY - visibleLines + 1
	}

	return nil
}

// View renders the editor panel.
func (e *EditorPanel) View() string {
	var b strings.Builder

	title := PanelTitleStyle.Render("Query Editor")
	hint := MutedText.Render(" Ctrl+Enter to run")
	b.WriteString(title + hint + "\n")

	visibleLines := e.height - 4
	if visibleLines < 1 {
		visibleLines = 1
	}

	end := e.scrollY + visibleLines
	if end > len(e.lines) {
		end = len(e.lines)
	}

	for i := e.scrollY; i < end; i++ {
		lineNum := EditorLineNumber.Render(strings.Repeat(" ", 3-len(lipgloss.NewStyle().Render(string(rune('0'+i%10))))) + string(rune('0'+(i+1)%10)))
		_ = lineNum

		line := e.lines[i]

		// Render with cursor if active
		var rendered string
		if e.active && i == e.cursorY {
			if e.cursorX >= len(line) {
				rendered = line + EditorCursor.Render(" ")
			} else {
				before := line[:e.cursorX]
				cursor := EditorCursor.Render(string(line[e.cursorX]))
				after := line[e.cursorX+1:]
				rendered = before + cursor + after
			}
		} else {
			rendered = highlightSQL(line)
		}

		b.WriteString(MutedText.Render(padLeft(i+1, 3)) + " " + rendered + "\n")
	}

	return e.wrapInBorder(b.String())
}

func padLeft(n, width int) string {
	s := strings.Builder{}
	ns := intToStr(n)
	for i := 0; i < width-len(ns); i++ {
		s.WriteByte(' ')
	}
	s.WriteString(ns)
	return s.String()
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// highlightSQL applies basic syntax highlighting to SQL.
func highlightSQL(line string) string {
	keywords := []string{
		"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "CREATE",
		"DROP", "ALTER", "TABLE", "INTO", "VALUES", "SET", "JOIN", "LEFT",
		"RIGHT", "INNER", "OUTER", "ON", "AND", "OR", "NOT", "NULL",
		"IS", "IN", "LIKE", "BETWEEN", "ORDER", "BY", "GROUP", "HAVING",
		"LIMIT", "OFFSET", "AS", "DISTINCT", "COUNT", "SUM", "AVG",
		"MAX", "MIN", "DESC", "ASC", "UNION", "ALL", "EXISTS", "CASE",
		"WHEN", "THEN", "ELSE", "END", "BEGIN", "COMMIT", "ROLLBACK",
	}

	keywordStyle := lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true)

	words := strings.Fields(line)
	for i, w := range words {
		upper := strings.ToUpper(w)
		for _, kw := range keywords {
			if upper == kw {
				words[i] = keywordStyle.Render(w)
				break
			}
		}
	}
	return strings.Join(words, " ")
}

func (e *EditorPanel) wrapInBorder(content string) string {
	style := InactivePanelBorder
	if e.active {
		style = ActivePanelBorder
	}
	return style.Width(e.width).Height(e.height).Render(content)
}
