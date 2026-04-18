package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ExportFormat represents an export format.
type ExportFormat int

const (
	ExportCSV ExportFormat = iota
	ExportJSON
	ExportSQL
)

// ExportRequestMsg is sent when export is requested.
type ExportRequestMsg struct {
	Format   ExportFormat
	FilePath string
}

// ExportDialog allows the user to choose export format and path.
type ExportDialog struct {
	visible  bool
	cursor   int
	input    string
	editing  bool // Editing the file path
	width    int
	height   int
}

type exportOption struct {
	format ExportFormat
	name   string
	ext    string
}

var exportOptions = []exportOption{
	{ExportCSV, "CSV", ".csv"},
	{ExportJSON, "JSON", ".json"},
	{ExportSQL, "SQL INSERT", ".sql"},
}

// NewExportDialog creates a new export dialog.
func NewExportDialog() ExportDialog {
	return ExportDialog{}
}

// Show shows the export dialog.
func (e *ExportDialog) Show() {
	e.visible = true
	e.cursor = 0
	e.input = "export"
	e.editing = false
}

// Hide hides the export dialog.
func (e *ExportDialog) Hide() {
	e.visible = false
}

// IsVisible returns whether the dialog is visible.
func (e *ExportDialog) IsVisible() bool {
	return e.visible
}

// SetSize sets the dialog dimensions.
func (e *ExportDialog) SetSize(w, h int) {
	e.width = w
	e.height = h
}

// Update handles input for the export dialog.
func (e *ExportDialog) Update(msg tea.KeyMsg) tea.Cmd {
	if e.editing {
		switch msg.Type {
		case tea.KeyEsc:
			e.editing = false
		case tea.KeyEnter:
			opt := exportOptions[e.cursor]
			path := e.input + opt.ext
			e.Hide()
			return func() tea.Msg {
				return ExportRequestMsg{Format: opt.format, FilePath: path}
			}
		case tea.KeyBackspace:
			if len(e.input) > 0 {
				e.input = e.input[:len(e.input)-1]
			}
		case tea.KeyRunes:
			e.input += string(msg.Runes)
		}
		return nil
	}

	switch msg.Type {
	case tea.KeyEsc:
		e.Hide()
	case tea.KeyUp:
		if e.cursor > 0 {
			e.cursor--
		}
	case tea.KeyDown:
		if e.cursor < len(exportOptions)-1 {
			e.cursor++
		}
	case tea.KeyEnter:
		e.editing = true
	}
	return nil
}

// View renders the export dialog.
func (e *ExportDialog) View() string {
	if !e.visible {
		return ""
	}

	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render("  Export Results")
	b.WriteString(title + "\n\n")

	if e.editing {
		opt := exportOptions[e.cursor]
		b.WriteString("  Format: " + lipgloss.NewStyle().Bold(true).Foreground(SecondaryColor).Render(opt.name) + "\n\n")
		b.WriteString("  File name: " + e.input + EditorCursor.Render(" ") + opt.ext + "\n\n")
		b.WriteString(MutedText.Render("  Enter to export, Esc to go back"))
	} else {
		b.WriteString(MutedText.Render("  Select format:") + "\n\n")
		for i, opt := range exportOptions {
			line := "  " + opt.name + " (" + opt.ext + ")"
			if i == e.cursor {
				line = SelectedRow.Render(line)
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n" + MutedText.Render("  Enter to select, Esc to cancel"))
	}

	modalW := 40
	if modalW > e.width-4 {
		modalW = e.width - 4
	}

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Width(modalW).
		Padding(1, 2).
		Render(b.String())

	return lipgloss.Place(e.width, e.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}
