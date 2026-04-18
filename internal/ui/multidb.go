package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
)

// DBConnection represents a single active database connection.
type DBConnection struct {
	Config db.ConnectionConfig
	Driver db.Driver
	Name   string
	DBType string
}

// MultiDBManager manages multiple simultaneous database connections.
type MultiDBManager struct {
	connections []*DBConnection
	active      int
	width       int
}

// NewMultiDBManager creates a new multi-database manager.
func NewMultiDBManager() MultiDBManager {
	return MultiDBManager{}
}

// AddConnection adds a new database connection.
func (m *MultiDBManager) AddConnection(config db.ConnectionConfig, driver db.Driver) int {
	name := config.Name
	if name == "" {
		name = fmt.Sprintf("%s:%s", config.Type, config.Database)
	}

	conn := &DBConnection{
		Config: config,
		Driver: driver,
		Name:   name,
		DBType: driver.DatabaseType(),
	}
	m.connections = append(m.connections, conn)
	m.active = len(m.connections) - 1
	return m.active
}

// RemoveConnection disconnects and removes a connection.
func (m *MultiDBManager) RemoveConnection(idx int) {
	if idx < 0 || idx >= len(m.connections) {
		return
	}
	m.connections[idx].Driver.Disconnect()
	m.connections = append(m.connections[:idx], m.connections[idx+1:]...)
	if m.active >= len(m.connections) {
		m.active = len(m.connections) - 1
	}
}

// SetActive sets the active connection.
func (m *MultiDBManager) SetActive(idx int) {
	if idx >= 0 && idx < len(m.connections) {
		m.active = idx
	}
}

// NextConnection switches to the next connection.
func (m *MultiDBManager) NextConnection() {
	if len(m.connections) > 1 {
		m.active = (m.active + 1) % len(m.connections)
	}
}

// PrevConnection switches to the previous connection.
func (m *MultiDBManager) PrevConnection() {
	if len(m.connections) > 1 {
		m.active = (m.active - 1 + len(m.connections)) % len(m.connections)
	}
}

// ActiveConnection returns the currently active connection.
func (m *MultiDBManager) ActiveConnection() *DBConnection {
	if m.active >= 0 && m.active < len(m.connections) {
		return m.connections[m.active]
	}
	return nil
}

// ActiveDriver returns the driver for the active connection.
func (m *MultiDBManager) ActiveDriver() db.Driver {
	conn := m.ActiveConnection()
	if conn != nil {
		return conn.Driver
	}
	return nil
}

// Count returns the number of connections.
func (m *MultiDBManager) Count() int {
	return len(m.connections)
}

// SetWidth sets the tab bar width.
func (m *MultiDBManager) SetWidth(w int) {
	m.width = w
}

// DisconnectAll disconnects all connections.
func (m *MultiDBManager) DisconnectAll() {
	for _, conn := range m.connections {
		conn.Driver.Disconnect()
	}
	m.connections = nil
	m.active = -1
}

// View renders the connection tab bar.
func (m *MultiDBManager) View() string {
	if len(m.connections) <= 1 {
		return ""
	}

	var parts []string

	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(PrimaryColor).
		Padding(0, 1)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(MutedColor).
		Padding(0, 1)

	for i, conn := range m.connections {
		icon := dbIcon(conn.DBType)
		label := icon + " " + conn.Name

		if i == m.active {
			parts = append(parts, activeStyle.Render(label))
		} else {
			parts = append(parts, inactiveStyle.Render(label))
		}
	}

	return strings.Join(parts, "") + "\n"
}

func dbIcon(dbType string) string {
	switch dbType {
	case "PostgreSQL":
		return "PG"
	case "MySQL":
		return "MY"
	case "SQLite":
		return "SQ"
	case "MongoDB":
		return "MG"
	case "Redis":
		return "RD"
	default:
		return "DB"
	}
}
