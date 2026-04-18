package ui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
)

// DataGenRequestMsg is sent when the user confirms fake data generation.
type DataGenRequestMsg struct {
	Table string
	Count int
	SQL   string
}

// DataGenDialog shows a form for generating fake data into a table.
type DataGenDialog struct {
	visible   bool
	tableName string
	columns   []db.Column
	countStr  string
	cursor    int // 0 = count field, 1 = generate button
	width     int
	height    int
}

// NewDataGenDialog creates a new data generation dialog.
func NewDataGenDialog() DataGenDialog {
	return DataGenDialog{}
}

// Show shows the data generation dialog for the given table.
func (d *DataGenDialog) Show(tableName string, columns []db.Column) {
	d.visible = true
	d.tableName = tableName
	d.columns = columns
	d.countStr = "100"
	d.cursor = 0
}

// Hide hides the dialog.
func (d *DataGenDialog) Hide() {
	d.visible = false
}

// IsVisible returns whether the dialog is visible.
func (d *DataGenDialog) IsVisible() bool {
	return d.visible
}

// SetSize sets the dialog dimensions.
func (d *DataGenDialog) SetSize(w, h int) {
	d.width = w
	d.height = h
}

// Update handles input for the data generation dialog.
func (d *DataGenDialog) Update(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEsc:
		d.Hide()
		return nil
	case tea.KeyTab, tea.KeyDown:
		d.cursor = (d.cursor + 1) % 2
		return nil
	case tea.KeyShiftTab, tea.KeyUp:
		d.cursor = (d.cursor + 1) % 2
		return nil
	case tea.KeyEnter:
		if d.cursor == 1 {
			count := d.parseCount()
			sql := d.generateSQL(count)
			d.Hide()
			return func() tea.Msg {
				return DataGenRequestMsg{
					Table: d.tableName,
					Count: count,
					SQL:   sql,
				}
			}
		}
		return nil
	case tea.KeyBackspace:
		if d.cursor == 0 && len(d.countStr) > 0 {
			d.countStr = d.countStr[:len(d.countStr)-1]
		}
		return nil
	case tea.KeyRunes:
		if d.cursor == 0 {
			for _, r := range msg.Runes {
				if r >= '0' && r <= '9' {
					d.countStr += string(r)
				}
			}
		}
		return nil
	}
	return nil
}

// View renders the data generation dialog.
func (d *DataGenDialog) View() string {
	if !d.visible {
		return ""
	}

	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render("  Generate Fake Data")
	b.WriteString(title + "\n\n")

	// Table name (read-only)
	b.WriteString("  Table: " + lipgloss.NewStyle().Bold(true).Foreground(SecondaryColor).Render(d.tableName) + "\n\n")

	// Column summary
	b.WriteString(MutedText.Render("  Columns:") + "\n")
	for _, col := range d.columns {
		marker := "  "
		if col.PrimaryKey {
			marker = " *"
		}
		b.WriteString(MutedText.Render(fmt.Sprintf("  %s %-15s %s", marker, col.Name, col.DataType)) + "\n")
	}
	b.WriteString("\n")

	// Row count input
	countLabel := "  Rows: "
	countValue := d.countStr + " "
	if d.cursor == 0 {
		countValue = d.countStr + EditorCursor.Render(" ")
		countLabel = lipgloss.NewStyle().Foreground(AccentColor).Render(countLabel)
	}
	b.WriteString(countLabel + countValue + "\n\n")

	// Generate button
	btnText := "  [ Generate ]  "
	if d.cursor == 1 {
		btnText = SelectedRow.Render(btnText)
	} else {
		btnText = MutedText.Render(btnText)
	}
	b.WriteString("  " + btnText + "\n\n")

	b.WriteString(MutedText.Render("  Tab to switch fields, Enter to generate, Esc to cancel"))

	modalW := 50
	if modalW > d.width-4 {
		modalW = d.width - 4
	}

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Width(modalW).
		Padding(1, 2).
		Render(b.String())

	return lipgloss.Place(d.width, d.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

func (d *DataGenDialog) parseCount() int {
	count := 0
	for _, r := range d.countStr {
		if r >= '0' && r <= '9' {
			count = count*10 + int(r-'0')
		}
	}
	if count <= 0 {
		count = 100
	}
	if count > 10000 {
		count = 10000
	}
	return count
}

func (d *DataGenDialog) generateSQL(count int) string {
	if len(d.columns) == 0 {
		return ""
	}

	// Filter out auto-increment primary key columns
	var insertCols []db.Column
	for _, col := range d.columns {
		extra := strings.ToLower(col.Extra)
		if strings.Contains(extra, "auto_increment") || strings.Contains(extra, "autoincrement") {
			continue
		}
		insertCols = append(insertCols, col)
	}
	if len(insertCols) == 0 {
		insertCols = d.columns
	}

	// Build column name list
	colNames := make([]string, len(insertCols))
	for i, col := range insertCols {
		colNames[i] = col.Name
	}

	var sb strings.Builder
	src := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < count; i++ {
		values := make([]string, len(insertCols))
		for j, col := range insertCols {
			values[j] = generateValue(col, src)
		}
		sb.WriteString(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);\n",
			d.tableName,
			strings.Join(colNames, ", "),
			strings.Join(values, ", "),
		))
	}
	return sb.String()
}

// Word lists for fake text generation.
var (
	fakeWords = []string{
		"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf",
		"hotel", "india", "juliet", "kilo", "lima", "mike", "november",
		"oscar", "papa", "quebec", "romeo", "sierra", "tango", "uniform",
		"victor", "whiskey", "xray", "yankee", "zulu",
	}
	fakeFirstNames = []string{
		"Alice", "Bob", "Carol", "Dave", "Eve", "Frank", "Grace",
		"Hank", "Iris", "Jack", "Karen", "Leo", "Mona", "Nick",
		"Olivia", "Paul", "Quinn", "Rosa", "Sam", "Tina",
	}
	fakeLastNames = []string{
		"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia",
		"Miller", "Davis", "Rodriguez", "Martinez", "Anderson", "Taylor",
		"Thomas", "Jackson", "White", "Harris", "Martin", "Clark",
	}
)

func generateValue(col db.Column, src *rand.Rand) string {
	dt := strings.ToUpper(col.DataType)

	switch {
	case strings.Contains(dt, "INT") || strings.Contains(dt, "INTEGER"):
		return fmt.Sprintf("%d", src.Intn(100000))

	case strings.Contains(dt, "REAL") || strings.Contains(dt, "FLOAT") ||
		strings.Contains(dt, "DOUBLE") || strings.Contains(dt, "DECIMAL") ||
		strings.Contains(dt, "NUMERIC"):
		return fmt.Sprintf("%.2f", src.Float64()*10000)

	case strings.Contains(dt, "BOOL"):
		if src.Intn(2) == 0 {
			return "0"
		}
		return "1"

	case strings.Contains(dt, "DATE") || strings.Contains(dt, "TIMESTAMP"):
		now := time.Now()
		daysBack := src.Intn(730) // last 2 years
		t := now.AddDate(0, 0, -daysBack)
		if strings.Contains(dt, "TIME") || strings.Contains(dt, "TIMESTAMP") {
			t = t.Add(time.Duration(src.Intn(86400)) * time.Second)
			return fmt.Sprintf("'%s'", t.Format("2006-01-02 15:04:05"))
		}
		return fmt.Sprintf("'%s'", t.Format("2006-01-02"))

	case strings.Contains(dt, "TEXT") || strings.Contains(dt, "VARCHAR") ||
		strings.Contains(dt, "CHAR") || strings.Contains(dt, "STRING"):
		name := col.Name
		lower := strings.ToLower(name)
		switch {
		case strings.Contains(lower, "email"):
			first := fakeFirstNames[src.Intn(len(fakeFirstNames))]
			last := fakeLastNames[src.Intn(len(fakeLastNames))]
			return fmt.Sprintf("'%s.%s@example.com'", strings.ToLower(first), strings.ToLower(last))
		case strings.Contains(lower, "name"):
			first := fakeFirstNames[src.Intn(len(fakeFirstNames))]
			last := fakeLastNames[src.Intn(len(fakeLastNames))]
			return fmt.Sprintf("'%s %s'", first, last)
		default:
			nWords := 2 + src.Intn(4)
			words := make([]string, nWords)
			for i := range words {
				words[i] = fakeWords[src.Intn(len(fakeWords))]
			}
			return fmt.Sprintf("'%s'", strings.Join(words, " "))
		}

	default:
		// Fallback: generate a text value
		word := fakeWords[src.Intn(len(fakeWords))]
		return fmt.Sprintf("'%s'", word)
	}
}
