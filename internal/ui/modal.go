package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Modal renders a centered modal dialog.
type Modal struct {
	title   string
	content string
	width   int
	height  int
	visible bool
	screenW int
	screenH int
}

// NewModal creates a new modal.
func NewModal() Modal {
	return Modal{}
}

// Show shows the modal with given content.
func (m *Modal) Show(title, content string) {
	m.title = title
	m.content = content
	m.visible = true
}

// Hide hides the modal.
func (m *Modal) Hide() {
	m.visible = false
}

// IsVisible returns whether the modal is visible.
func (m *Modal) IsVisible() bool {
	return m.visible
}

// SetScreenSize sets the screen dimensions for centering.
func (m *Modal) SetScreenSize(w, h int) {
	m.screenW = w
	m.screenH = h
}

// View renders the modal.
func (m *Modal) View() string {
	if !m.visible {
		return ""
	}

	modalW := m.screenW / 2
	if modalW < 40 {
		modalW = 40
	}
	modalH := m.screenH / 2
	if modalH < 10 {
		modalH = 10
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render(m.title)
	body := title + "\n\n" + m.content + "\n\n" + MutedText.Render("Press Escape to close")

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Width(modalW).
		Height(modalH).
		Padding(1, 2).
		Render(body)

	return lipgloss.Place(m.screenW, m.screenH,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}
