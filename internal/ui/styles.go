package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	PrimaryColor   = lipgloss.Color("#7C3AED") // Purple
	SecondaryColor = lipgloss.Color("#06B6D4") // Cyan
	AccentColor    = lipgloss.Color("#F59E0B") // Amber
	SuccessColor   = lipgloss.Color("#10B981") // Green
	ErrorColor     = lipgloss.Color("#EF4444") // Red
	MutedColor     = lipgloss.Color("#6B7280") // Gray
	TextColor      = lipgloss.Color("#E5E7EB") // Light gray
	BgColor        = lipgloss.Color("#1F2937") // Dark
	PanelBgColor   = lipgloss.Color("#111827") // Darker

	// Panel styles
	ActivePanelBorder = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(PrimaryColor)

	InactivePanelBorder = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(MutedColor)

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor).
			Padding(0, 1)

	PanelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(SecondaryColor)

	// Text styles
	NormalText = lipgloss.NewStyle().
			Foreground(TextColor)

	MutedText = lipgloss.NewStyle().
			Foreground(MutedColor)

	SelectedRow = lipgloss.NewStyle().
			Background(PrimaryColor).
			Foreground(lipgloss.Color("#FFFFFF"))

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#374151")).
			Foreground(TextColor).
			Padding(0, 1)

	StatusBarKeyStyle = lipgloss.NewStyle().
				Background(PrimaryColor).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Padding(0, 1)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(SecondaryColor).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(MutedColor)

	NullStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Italic(true)

	// Tree styles
	TreeNodeStyle = lipgloss.NewStyle().
			Foreground(TextColor)

	TreeNodeSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(PrimaryColor)

	TreeBranchStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	// Editor styles
	EditorLineNumber = lipgloss.NewStyle().
				Foreground(MutedColor).
				Width(4).
				Align(lipgloss.Right)

	EditorCursor = lipgloss.NewStyle().
			Background(TextColor).
			Foreground(PanelBgColor)

	// Help styles
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(AccentColor).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(MutedColor)
)
