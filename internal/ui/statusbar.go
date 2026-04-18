package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StatusBar displays connection info and status at the bottom.
type StatusBar struct {
	dbType     string
	dbVersion  string
	connString string
	queryTime  string
	rowCount   int
	width      int
	message    string
}

// NewStatusBar creates a new status bar.
func NewStatusBar() StatusBar {
	return StatusBar{}
}

// SetWidth sets the status bar width.
func (s *StatusBar) SetWidth(w int) {
	s.width = w
}

// SetConnection sets the connection info.
func (s *StatusBar) SetConnection(dbType, version, connString string) {
	s.dbType = dbType
	s.dbVersion = version
	s.connString = connString
}

// SetQueryInfo sets the query timing and row count.
func (s *StatusBar) SetQueryInfo(duration string, rows int) {
	s.queryTime = duration
	s.rowCount = rows
}

// SetMessage sets a temporary status message.
func (s *StatusBar) SetMessage(msg string) {
	s.message = msg
}

// View renders the status bar.
func (s *StatusBar) View() string {
	left := StatusBarKeyStyle.Render(" LazyDB ")

	if s.dbType != "" {
		connInfo := fmt.Sprintf(" Connected: %s", s.dbType)
		if s.dbVersion != "" {
			connInfo += " " + s.dbVersion
		}
		left += StatusBarStyle.Render(connInfo)
	}

	var right string
	if s.message != "" {
		right = StatusBarStyle.Render(s.message)
	} else {
		var parts []string
		if s.queryTime != "" {
			parts = append(parts, fmt.Sprintf("Query: %s", s.queryTime))
		}
		if s.rowCount > 0 {
			parts = append(parts, fmt.Sprintf("%d rows", s.rowCount))
		}
		parts = append(parts, "? Help")
		right = StatusBarStyle.Render(strings.Join(parts, " │ "))
	}

	// Fill the gap between left and right
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := s.width - leftW - rightW
	if gap < 0 {
		gap = 0
	}
	filler := StatusBarStyle.Render(strings.Repeat(" ", gap))

	return left + filler + right
}
