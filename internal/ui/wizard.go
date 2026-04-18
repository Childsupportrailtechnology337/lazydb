package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
)

// WizardResultMsg is sent when the user completes the wizard.
type WizardResultMsg struct {
	Config db.ConnectionConfig
}

// WizardCancelMsg is sent when the user cancels the wizard.
type WizardCancelMsg struct{}

// TestConnectionMsg is sent when the user requests a connection test.
type TestConnectionMsg struct {
	Config db.ConnectionConfig
}

// TestConnectionResultMsg carries the result of a connection test.
type TestConnectionResultMsg struct {
	Success bool
	Error   string
}

// Database type constants.
const (
	dbPostgres = "postgres"
	dbMySQL    = "mysql"
	dbSQLite   = "sqlite"
	dbMongoDB  = "mongodb"
	dbRedis    = "redis"
)

// wizardStep tracks which section of the wizard is focused.
type wizardStep int

const (
	stepDBType wizardStep = iota
	stepForm
	stepSSHTunnel
	stepActions
)

// actionIndex tracks which action button is focused.
type actionIndex int

const (
	actionTest actionIndex = iota
	actionSaveProfile
	actionConnect
)

// dbTypeOption represents a selectable database type.
type dbTypeOption struct {
	label string
	value string
	port  int
}

var dbTypes = []dbTypeOption{
	{label: "PostgreSQL", value: dbPostgres, port: 5432},
	{label: "MySQL", value: dbMySQL, port: 3306},
	{label: "SQLite", value: dbSQLite, port: 0},
	{label: "MongoDB", value: dbMongoDB, port: 27017},
	{label: "Redis", value: dbRedis, port: 6379},
}

// formField represents an input field in the form.
type formField struct {
	label       string
	value       string
	placeholder string
	cursorPos   int
	hidden      bool // for password fields
}

// WizardModel is the Bubble Tea model for the connection wizard.
type WizardModel struct {
	width  int
	height int

	// Database type selection
	dbTypeIndex int

	// Form fields (rebuilt when db type changes)
	fields     []formField
	fieldIndex int

	// SSH tunnel
	sshEnabled  bool
	sshFields   []formField
	sshFieldIdx int

	// Actions
	activeAction actionIndex
	saveProfile  bool
	profileName  string

	// Current wizard step
	step wizardStep

	// Connection test status
	testStatus  string
	testSuccess bool

	// Status message
	statusMsg string
}

// NewWizardModel creates a new connection wizard.
func NewWizardModel() WizardModel {
	m := WizardModel{
		step:        stepDBType,
		dbTypeIndex: 0,
		sshFields:   makeSSHFields(),
	}
	m.fields = m.buildFormFields()
	return m
}

func makeSSHFields() []formField {
	return []formField{
		{label: "SSH Host", placeholder: "ssh.example.com"},
		{label: "SSH Port", value: "22", placeholder: "22"},
		{label: "SSH User", placeholder: "root"},
		{label: "SSH Key Path", placeholder: "~/.ssh/id_rsa"},
	}
}

func (m *WizardModel) buildFormFields() []formField {
	switch dbTypes[m.dbTypeIndex].value {
	case dbPostgres, dbMySQL:
		port := strconv.Itoa(dbTypes[m.dbTypeIndex].port)
		return []formField{
			{label: "Host", value: "localhost", placeholder: "localhost"},
			{label: "Port", value: port, placeholder: port},
			{label: "User", placeholder: "username"},
			{label: "Password", placeholder: "password", hidden: true},
			{label: "Database", placeholder: "mydb"},
			{label: "SSL Mode", value: "disable", placeholder: "disable / require / verify-full"},
		}
	case dbSQLite:
		return []formField{
			{label: "File Path", placeholder: "/path/to/database.db"},
		}
	case dbMongoDB:
		return []formField{
			{label: "URI", placeholder: "mongodb://localhost:27017"},
			{label: "Host", value: "localhost", placeholder: "localhost"},
			{label: "Port", value: "27017", placeholder: "27017"},
			{label: "Database", placeholder: "mydb"},
		}
	case dbRedis:
		return []formField{
			{label: "Host", value: "localhost", placeholder: "localhost"},
			{label: "Port", value: "6379", placeholder: "6379"},
			{label: "Password", placeholder: "password", hidden: true},
			{label: "Database", value: "0", placeholder: "0"},
		}
	}
	return nil
}

func (m *WizardModel) buildConfig() db.ConnectionConfig {
	cfg := db.ConnectionConfig{
		Type: dbTypes[m.dbTypeIndex].value,
	}
	if m.saveProfile && m.profileName != "" {
		cfg.Name = m.profileName
	}

	switch cfg.Type {
	case dbPostgres, dbMySQL:
		cfg.Host = m.fields[0].value
		cfg.Port, _ = strconv.Atoi(m.fields[1].value)
		cfg.User = m.fields[2].value
		cfg.Password = m.fields[3].value
		cfg.Database = m.fields[4].value
		cfg.SSLMode = m.fields[5].value
	case dbSQLite:
		cfg.FilePath = m.fields[0].value
	case dbMongoDB:
		cfg.URI = m.fields[0].value
		cfg.Host = m.fields[1].value
		cfg.Port, _ = strconv.Atoi(m.fields[2].value)
		cfg.Database = m.fields[3].value
	case dbRedis:
		cfg.Host = m.fields[0].value
		cfg.Port, _ = strconv.Atoi(m.fields[1].value)
		cfg.Password = m.fields[2].value
		cfg.Database = m.fields[3].value
	}

	if m.sshEnabled {
		cfg.SSHHost = m.sshFields[0].value
		cfg.SSHPort = m.sshFields[1].value
		cfg.SSHUser = m.sshFields[2].value
		cfg.SSHKeyPath = m.sshFields[3].value
	}

	return cfg
}

// SetSize sets the terminal dimensions.
func (m *WizardModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Init implements tea.Model.
func (m WizardModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m WizardModel) Update(msg tea.Msg) (WizardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case TestConnectionResultMsg:
		if msg.Success {
			m.testStatus = "Connection successful!"
			m.testSuccess = true
		} else {
			m.testStatus = "Connection failed: " + msg.Error
			m.testSuccess = false
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m WizardModel) handleKey(msg tea.KeyMsg) (WizardModel, tea.Cmd) {
	key := msg.String()

	// Global keys
	switch key {
	case "esc":
		return m, func() tea.Msg { return WizardCancelMsg{} }
	case "tab":
		m.advanceStep()
		return m, nil
	case "shift+tab":
		m.retreatStep()
		return m, nil
	}

	switch m.step {
	case stepDBType:
		return m.handleDBTypeKey(key)
	case stepForm:
		return m.handleFormKey(msg)
	case stepSSHTunnel:
		return m.handleSSHKey(msg)
	case stepActions:
		return m.handleActionKey(key)
	}
	return m, nil
}

func (m *WizardModel) advanceStep() {
	switch m.step {
	case stepDBType:
		m.step = stepForm
		m.fieldIndex = 0
	case stepForm:
		if dbTypes[m.dbTypeIndex].value != dbSQLite {
			m.step = stepSSHTunnel
		} else {
			m.step = stepActions
		}
	case stepSSHTunnel:
		m.step = stepActions
		m.activeAction = actionTest
	case stepActions:
		m.step = stepDBType
	}
}

func (m *WizardModel) retreatStep() {
	switch m.step {
	case stepDBType:
		m.step = stepActions
	case stepForm:
		m.step = stepDBType
	case stepSSHTunnel:
		m.step = stepForm
		m.fieldIndex = len(m.fields) - 1
	case stepActions:
		if dbTypes[m.dbTypeIndex].value != dbSQLite {
			m.step = stepSSHTunnel
		} else {
			m.step = stepForm
			m.fieldIndex = len(m.fields) - 1
		}
	}
}

func (m WizardModel) handleDBTypeKey(key string) (WizardModel, tea.Cmd) {
	switch key {
	case "j", "down":
		m.dbTypeIndex++
		if m.dbTypeIndex >= len(dbTypes) {
			m.dbTypeIndex = 0
		}
		m.fields = m.buildFormFields()
		m.fieldIndex = 0
		m.testStatus = ""
	case "k", "up":
		m.dbTypeIndex--
		if m.dbTypeIndex < 0 {
			m.dbTypeIndex = len(dbTypes) - 1
		}
		m.fields = m.buildFormFields()
		m.fieldIndex = 0
		m.testStatus = ""
	case "enter":
		m.step = stepForm
		m.fieldIndex = 0
	}
	return m, nil
}

func (m WizardModel) handleFormKey(msg tea.KeyMsg) (WizardModel, tea.Cmd) {
	key := msg.String()
	switch key {
	case "up":
		if m.fieldIndex > 0 {
			m.fieldIndex--
		}
		return m, nil
	case "down", "enter":
		if m.fieldIndex < len(m.fields)-1 {
			m.fieldIndex++
		} else if key == "enter" {
			m.advanceStep()
		}
		return m, nil
	}
	m.editField(&m.fields[m.fieldIndex], msg)
	return m, nil
}

func (m WizardModel) handleSSHKey(msg tea.KeyMsg) (WizardModel, tea.Cmd) {
	key := msg.String()

	// Toggle SSH on/off with space when at the toggle position
	if !m.sshEnabled {
		if key == " " || key == "enter" {
			m.sshEnabled = true
			m.sshFieldIdx = 0
		}
		return m, nil
	}

	switch key {
	case "up":
		if m.sshFieldIdx > 0 {
			m.sshFieldIdx--
		}
		return m, nil
	case "down", "enter":
		if m.sshFieldIdx < len(m.sshFields)-1 {
			m.sshFieldIdx++
		} else if key == "enter" {
			m.advanceStep()
		}
		return m, nil
	case "ctrl+d":
		m.sshEnabled = false
		return m, nil
	}
	m.editField(&m.sshFields[m.sshFieldIdx], msg)
	return m, nil
}

func (m WizardModel) handleActionKey(key string) (WizardModel, tea.Cmd) {
	switch key {
	case "left", "h":
		if m.activeAction > 0 {
			m.activeAction--
		}
	case "right", "l":
		if m.activeAction < actionConnect {
			m.activeAction++
		}
	case "enter", " ":
		switch m.activeAction {
		case actionTest:
			cfg := m.buildConfig()
			m.testStatus = "Testing connection..."
			m.testSuccess = false
			return m, func() tea.Msg { return TestConnectionMsg{Config: cfg} }
		case actionSaveProfile:
			m.saveProfile = !m.saveProfile
			return m, nil
		case actionConnect:
			cfg := m.buildConfig()
			return m, func() tea.Msg { return WizardResultMsg{Config: cfg} }
		}
	}
	return m, nil
}

func (m *WizardModel) editField(f *formField, msg tea.KeyMsg) {
	key := msg.String()
	switch key {
	case "backspace":
		if len(f.value) > 0 {
			f.value = f.value[:len(f.value)-1]
		}
	case "ctrl+u":
		f.value = ""
	default:
		// Only accept printable characters
		if len(key) == 1 && key[0] >= 32 && key[0] <= 126 {
			f.value += key
		}
	}
}

// View implements tea.Model.
func (m WizardModel) View() string {
	if m.width == 0 {
		return ""
	}

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(PrimaryColor).
		MarginBottom(1).
		Render("  LazyDB Connection Wizard")

	// Build sections
	dbSection := m.viewDBTypeSection()
	formSection := m.viewFormSection()
	sshSection := m.viewSSHSection()
	actionSection := m.viewActionSection()

	// Status line
	statusLine := ""
	if m.testStatus != "" {
		color := ErrorColor
		if m.testSuccess {
			color = SuccessColor
		}
		statusLine = lipgloss.NewStyle().Foreground(color).Render(m.testStatus)
	}

	// Help
	help := m.viewHelp()

	// Compose body
	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		dbSection,
		"",
		formSection,
		"",
		sshSection,
		"",
		actionSection,
	)
	if statusLine != "" {
		body = lipgloss.JoinVertical(lipgloss.Left, body, "", statusLine)
	}
	body = lipgloss.JoinVertical(lipgloss.Left, body, "", help)

	// Outer frame
	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}
	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Width(contentWidth).
		Padding(1, 2).
		Render(body)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		frame,
	)
}

func (m *WizardModel) viewDBTypeSection() string {
	isActive := m.step == stepDBType
	titleStyle := PanelTitleStyle
	if isActive {
		titleStyle = titleStyle.Underline(true)
	}
	header := titleStyle.Render("Database Type")

	var items []string
	for i, dt := range dbTypes {
		radio := "  "
		if i == m.dbTypeIndex {
			radio = lipgloss.NewStyle().Foreground(SuccessColor).Render("● ")
		} else {
			radio = lipgloss.NewStyle().Foreground(MutedColor).Render("○ ")
		}
		label := dt.label
		if i == m.dbTypeIndex && isActive {
			label = lipgloss.NewStyle().
				Bold(true).
				Foreground(PrimaryColor).
				Render(label)
		} else {
			label = NormalText.Render(label)
		}
		items = append(items, radio+label)
	}

	content := strings.Join(items, "\n")
	return header + "\n" + content
}

func (m *WizardModel) viewFormSection() string {
	isActive := m.step == stepForm
	titleStyle := PanelTitleStyle
	if isActive {
		titleStyle = titleStyle.Underline(true)
	}
	header := titleStyle.Render("Connection Details")

	labelWidth := 0
	for _, f := range m.fields {
		if len(f.label) > labelWidth {
			labelWidth = len(f.label)
		}
	}
	labelWidth += 2

	var rows []string
	for i, f := range m.fields {
		rows = append(rows, m.renderField(f, i, m.fieldIndex, isActive, labelWidth))
	}

	content := strings.Join(rows, "\n")
	return header + "\n" + content
}

func (m *WizardModel) renderField(f formField, idx, activeIdx int, sectionActive bool, labelWidth int) string {
	isFocused := sectionActive && idx == activeIdx

	labelStyle := lipgloss.NewStyle().
		Width(labelWidth).
		Foreground(MutedColor)

	if isFocused {
		labelStyle = labelStyle.Foreground(AccentColor).Bold(true)
	}
	label := labelStyle.Render(f.label + ":")

	inputWidth := 40
	displayValue := f.value
	if f.hidden && displayValue != "" {
		displayValue = strings.Repeat("*", len(displayValue))
	}

	var input string
	if isFocused {
		cursor := lipgloss.NewStyle().Background(TextColor).Foreground(PanelBgColor).Render(" ")
		inputContent := displayValue + cursor
		if displayValue == "" {
			inputContent = lipgloss.NewStyle().Foreground(MutedColor).Render(f.placeholder) + cursor
		}
		input = lipgloss.NewStyle().
			Width(inputWidth).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(PrimaryColor).
			Render(inputContent)
	} else {
		content := displayValue
		if content == "" {
			content = lipgloss.NewStyle().Foreground(MutedColor).Italic(true).Render(f.placeholder)
		} else {
			content = NormalText.Render(content)
		}
		input = lipgloss.NewStyle().
			Width(inputWidth).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(MutedColor).
			Render(content)
	}

	return label + " " + input
}

func (m *WizardModel) viewSSHSection() string {
	if dbTypes[m.dbTypeIndex].value == dbSQLite {
		return MutedText.Render("  SSH Tunnel: not available for SQLite")
	}

	isActive := m.step == stepSSHTunnel
	titleStyle := PanelTitleStyle
	if isActive {
		titleStyle = titleStyle.Underline(true)
	}
	header := titleStyle.Render("SSH Tunnel")

	// Toggle
	toggleIcon := "○"
	toggleLabel := "Disabled"
	toggleColor := MutedColor
	if m.sshEnabled {
		toggleIcon = "●"
		toggleLabel = "Enabled"
		toggleColor = SuccessColor
	}
	toggle := lipgloss.NewStyle().Foreground(toggleColor).Render(toggleIcon+" "+toggleLabel)
	if isActive && !m.sshEnabled {
		toggle += MutedText.Render("  (press Space to enable)")
	}

	result := header + "\n" + toggle

	if m.sshEnabled {
		labelWidth := 14

		var rows []string
		for i, f := range m.sshFields {
			rows = append(rows, m.renderField(f, i, m.sshFieldIdx, isActive, labelWidth))
		}
		if isActive {
			rows = append(rows, MutedText.Render("  Ctrl+D to disable SSH"))
		}
		result += "\n" + strings.Join(rows, "\n")
	}

	return result
}

func (m *WizardModel) viewActionSection() string {
	isActive := m.step == stepActions

	// Test button
	testStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(MutedColor).
		Foreground(TextColor)
	if isActive && m.activeAction == actionTest {
		testStyle = testStyle.
			BorderForeground(AccentColor).
			Foreground(AccentColor).
			Bold(true)
	}
	testBtn := testStyle.Render("Test Connection")

	// Save profile checkbox
	checkIcon := "☐"
	if m.saveProfile {
		checkIcon = "☑"
	}
	saveStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(TextColor)
	if isActive && m.activeAction == actionSaveProfile {
		saveStyle = saveStyle.Foreground(AccentColor).Bold(true)
	}
	saveBtn := saveStyle.Render(fmt.Sprintf("%s Save Profile", checkIcon))

	// Connect button
	connectStyle := lipgloss.NewStyle().
		Padding(0, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(MutedColor).
		Foreground(TextColor)
	if isActive && m.activeAction == actionConnect {
		connectStyle = connectStyle.
			BorderForeground(SuccessColor).
			Foreground(SuccessColor).
			Bold(true)
	}
	connectBtn := connectStyle.Render("Connect")

	return lipgloss.JoinHorizontal(lipgloss.Center,
		testBtn,
		"  ",
		saveBtn,
		"  ",
		connectBtn,
	)
}

func (m *WizardModel) viewHelp() string {
	var pairs []string

	switch m.step {
	case stepDBType:
		pairs = []string{
			"↑/↓", "select type",
			"Tab", "next section",
			"Enter", "confirm",
			"Esc", "quit",
		}
	case stepForm:
		pairs = []string{
			"↑/↓", "move fields",
			"Tab", "next section",
			"Shift+Tab", "prev section",
			"Esc", "quit",
		}
	case stepSSHTunnel:
		pairs = []string{
			"Space", "toggle SSH",
			"Tab", "next section",
			"Shift+Tab", "prev section",
			"Esc", "quit",
		}
	case stepActions:
		pairs = []string{
			"←/→", "select action",
			"Enter", "activate",
			"Tab", "next section",
			"Esc", "quit",
		}
	}

	var rendered []string
	for i := 0; i < len(pairs); i += 2 {
		k := HelpKeyStyle.Render(pairs[i])
		d := HelpDescStyle.Render(pairs[i+1])
		rendered = append(rendered, k+" "+d)
	}
	return strings.Join(rendered, "    ")
}
