package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HelpOverlay renders the help modal.
type HelpOverlay struct {
	visible bool
	width   int
	height  int
}

// NewHelpOverlay creates a new help overlay.
func NewHelpOverlay() HelpOverlay {
	return HelpOverlay{}
}

// Toggle toggles the help overlay visibility.
func (h *HelpOverlay) Toggle() {
	h.visible = !h.visible
}

// IsVisible returns whether the overlay is visible.
func (h *HelpOverlay) IsVisible() bool {
	return h.visible
}

// SetSize sets the overlay dimensions.
func (h *HelpOverlay) SetSize(w, he int) {
	h.width = w
	h.height = he
}

type helpEntry struct {
	key  string
	desc string
}

// View renders the help overlay.
func (h *HelpOverlay) View() string {
	if !h.visible {
		return ""
	}

	sections := []struct {
		title   string
		entries []helpEntry
	}{
		{
			title: "General",
			entries: []helpEntry{
				{"?", "Toggle this help"},
				{"q / Ctrl+C", "Quit"},
				{"Tab", "Switch panels"},
				{"1 / 2 / 3", "Jump to panel"},
			},
		},
		{
			title: "Schema Browser",
			entries: []helpEntry{
				{"j / k", "Navigate up/down"},
				{"Enter", "Expand/collapse"},
				{"Space", "Preview table"},
				{"g / G", "Go to top/bottom"},
			},
		},
		{
			title: "Query Editor",
			entries: []helpEntry{
				{"Ctrl+Enter", "Execute query"},
				{"Alt+Up/Down", "Query history"},
				{"Enter", "New line"},
			},
		},
		{
			title: "Results",
			entries: []helpEntry{
				{"j / k", "Navigate rows"},
				{"h / l", "Scroll columns"},
				{"g / G", "Go to top/bottom"},
				{"n / p", "Next/prev page"},
			},
		},
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render("  LazyDB Help") + "\n\n")

	for _, section := range sections {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(SecondaryColor).Render("  "+section.title) + "\n")
		for _, e := range section.entries {
			key := HelpKeyStyle.Width(16).Render("  " + e.key)
			desc := HelpDescStyle.Render(e.desc)
			b.WriteString(key + desc + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(MutedText.Render("  Press ? or Escape to close"))

	modalW := 50
	modalH := 30
	if modalW > h.width-4 {
		modalW = h.width - 4
	}
	if modalH > h.height-4 {
		modalH = h.height - 4
	}

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Width(modalW).
		Height(modalH).
		Render(b.String())

	// Center the modal
	return lipgloss.Place(h.width, h.height,
		lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
	)
}

// PlaceOverlay places content on top of background.
func PlaceOverlay(x, y int, fg, bg string) string {
	fgLines := strings.Split(fg, "\n")
	bgLines := strings.Split(bg, "\n")

	for i, fgLine := range fgLines {
		bgIdx := y + i
		if bgIdx >= len(bgLines) {
			break
		}
		bgLine := bgLines[bgIdx]
		if x >= len(bgLine) {
			continue
		}
		fgWidth := lipgloss.Width(fgLine)
		_ = fgWidth
		bgLines[bgIdx] = bgLine[:x] + fgLine
	}
	return strings.Join(bgLines, "\n")
}
