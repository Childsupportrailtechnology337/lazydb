package app

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
	"github.com/aymenhmaidiwastaken/lazydb/internal/export"
	"github.com/aymenhmaidiwastaken/lazydb/internal/ui"
)

// Panel represents which panel is currently focused.
type Panel int

const (
	PanelSchema Panel = iota
	PanelEditor
	PanelResults
)

// Model is the main Bubble Tea model for LazyDB.
type Model struct {
	// Core
	driver     db.Driver
	config     db.ConnectionConfig
	multiDB    ui.MultiDBManager
	keyMap     *ui.KeyMap

	// Panels
	schema     ui.SchemaPanel
	editor     ui.EditorPanel
	results    ui.ResultsPanel
	statusBar  ui.StatusBar
	tabs       ui.QueryTabs

	// Overlays
	help       ui.HelpOverlay
	cmdPalette ui.CommandPalette
	rowDetail  ui.RowDetailPanel
	exportDlg  ui.ExportDialog
	txConfirm  ui.TxConfirmDialog

	// Features
	autoComp   ui.AutoComplete
	inlineEdit ui.InlineEditor
	txIndicator ui.TransactionIndicator
	fuzzyFilter ui.FuzzyFilter

	// State
	active     Panel
	width      int
	height     int
	err        error
	ready      bool
	lastResult *db.QueryResult
	loading    bool
	themeName  string
	searchMode bool
}

// New creates a new application model.
func New(config db.ConnectionConfig) Model {
	kbConfig := ui.LoadKeybindings()
	return Model{
		config:      config,
		multiDB:     ui.NewMultiDBManager(),
		keyMap:      ui.NewKeyMap(kbConfig),
		schema:      ui.NewSchemaPanel(),
		editor:      ui.NewEditorPanel(),
		results:     ui.NewResultsPanel(),
		statusBar:   ui.NewStatusBar(),
		tabs:        ui.NewQueryTabs(),
		help:        ui.NewHelpOverlay(),
		cmdPalette:  ui.NewCommandPalette(),
		rowDetail:   ui.NewRowDetailPanel(),
		exportDlg:   ui.NewExportDialog(),
		txConfirm:   ui.NewTxConfirmDialog(),
		autoComp:    ui.NewAutoComplete(),
		inlineEdit:  ui.NewInlineEditor(),
		txIndicator: ui.NewTransactionIndicator(),
		fuzzyFilter: ui.NewFuzzyFilter(),
		active:      PanelEditor,
		themeName:   "default",
	}
}

// NewWithTheme creates a model with a specific theme.
func NewWithTheme(config db.ConnectionConfig, theme string) Model {
	m := New(config)
	m.themeName = theme
	ui.ApplyTheme(ui.GetTheme(theme))
	return m
}

// Init initializes the model and starts the connection.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.connectCmd(),
		tea.EnterAltScreen,
	)
}

func (m Model) connectCmd() tea.Cmd {
	config := m.config
	return func() tea.Msg {
		driver, err := db.NewDriver(config.Type)
		if err != nil {
			return errMsg{err}
		}
		if err := driver.Connect(config); err != nil {
			return errMsg{err}
		}
		return connectedMsg{driver: driver, config: config}
	}
}

type connectedMsg struct {
	driver db.Driver
	config db.ConnectionConfig
}

type errMsg struct {
	err error
}

type queryResultMsg struct {
	result *db.QueryResult
	err    error
}

type exportDoneMsg struct {
	path string
	err  error
}

type clipboardMsg struct {
	text string
	err  error
}

// Update handles all events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.updateLayout()
		return m, nil

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case connectedMsg:
		m.driver = msg.driver
		m.multiDB.AddConnection(msg.config, msg.driver)
		m.schema.SetDriver(m.driver)
		m.statusBar.SetConnection(m.driver.DatabaseType(), "", msg.config.Name)
		m.updateLayout()
		return m, ui.LoadSchemaCmd(m.driver)

	case errMsg:
		m.err = msg.err
		m.loading = false
		m.statusBar.SetMessage(fmt.Sprintf("Error: %v", msg.err))
		return m, nil

	case ui.SchemaLoadedMsg:
		m.schema.HandleSchemaLoaded(msg)
		if msg.Err != nil {
			m.statusBar.SetMessage(fmt.Sprintf("Schema error: %v", msg.Err))
		} else {
			m.autoComp.SetSchemaFromNodes(msg.Nodes)
		}
		return m, nil

	case ui.ExecuteQueryMsg:
		m.editor.AddToHistory(msg.Query)
		m.tabs.SetActiveQuery(msg.Query)
		m.loading = true
		m.statusBar.SetMessage("Executing query...")
		if m.txIndicator.IsActive() {
			m.txIndicator.AddQuery()
		}
		return m, m.executeQuery(msg.Query)

	case queryResultMsg:
		m.loading = false
		if msg.err != nil {
			m.results.SetMessage(fmt.Sprintf("Error: %v", msg.err))
			m.statusBar.SetMessage(fmt.Sprintf("Query error: %v", msg.err))
		} else {
			m.results.SetResult(msg.result)
			m.lastResult = msg.result
			m.statusBar.SetQueryInfo(msg.result.Duration.String(), msg.result.RowCount)
		}
		return m, nil

	case ui.CommandPaletteMsg:
		return m.handleCommandAction(msg.Action)

	case ui.ExportRequestMsg:
		return m, m.doExport(msg)

	case exportDoneMsg:
		if msg.err != nil {
			m.statusBar.SetMessage(fmt.Sprintf("Export error: %v", msg.err))
		} else {
			m.statusBar.SetMessage(fmt.Sprintf("Exported to %s", msg.path))
		}
		return m, nil

	case ui.InlineEditMsg:
		// Execute the UPDATE
		m.statusBar.SetMessage(fmt.Sprintf("Executing: %s", msg.SQL))
		return m, m.executeQuery(msg.SQL)

	case ui.TransactionMsg:
		return m.handleTransaction(msg)

	case clipboardMsg:
		if msg.err != nil {
			m.statusBar.SetMessage(fmt.Sprintf("Clipboard error: %v", msg.err))
		} else {
			m.statusBar.SetMessage("Copied to clipboard")
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		// Determine which panel was clicked based on X position
		schemaWidth := m.width / 4
		if schemaWidth > 40 {
			schemaWidth = 40
		}

		if msg.X < schemaWidth {
			m.active = PanelSchema
		} else {
			editorHeight := (m.height - 3) / 2
			if msg.Y < editorHeight+1 {
				m.active = PanelEditor
			} else {
				m.active = PanelResults
			}
		}
		m.updatePanelActive()
	case tea.MouseButtonWheelUp:
		switch m.active {
		case PanelResults:
			m.results.Update(tea.KeyMsg{Type: tea.KeyUp})
		}
	case tea.MouseButtonWheelDown:
		switch m.active {
		case PanelResults:
			m.results.Update(tea.KeyMsg{Type: tea.KeyDown})
		}
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Overlays take priority
	if m.txConfirm.IsVisible() {
		return m.handleTxConfirmKey(msg)
	}
	if m.inlineEdit.IsActive() {
		cmd := m.inlineEdit.Update(msg)
		return m, cmd
	}
	if m.cmdPalette.IsVisible() {
		cmd := m.cmdPalette.Update(msg)
		return m, cmd
	}
	if m.rowDetail.IsVisible() {
		cmd := m.rowDetail.Update(msg)
		return m, cmd
	}
	if m.exportDlg.IsVisible() {
		cmd := m.exportDlg.Update(msg)
		return m, cmd
	}
	if m.help.IsVisible() {
		switch msg.String() {
		case "?", "esc", "q":
			m.help.Toggle()
		}
		return m, nil
	}

	// Schema filter mode
	if m.searchMode && m.active == PanelSchema {
		return m.handleSchemaSearch(msg)
	}

	// Global keybindings
	key := msg.String()
	switch key {
	case "ctrl+c":
		m.multiDB.DisconnectAll()
		return m, tea.Quit
	case "q":
		if m.active != PanelEditor {
			m.multiDB.DisconnectAll()
			return m, tea.Quit
		}
	case "?":
		if m.active != PanelEditor {
			m.help.Toggle()
			return m, nil
		}
	case ":":
		if m.active != PanelEditor {
			m.cmdPalette.Toggle()
			return m, nil
		}
	case "/":
		if m.active == PanelSchema {
			m.searchMode = true
			m.fuzzyFilter.SetActive(true)
			return m, nil
		}
	case "tab":
		m.cyclePanel(1)
		m.updatePanelActive()
		return m, nil
	case "shift+tab":
		m.cyclePanel(-1)
		m.updatePanelActive()
		return m, nil
	case "1":
		if m.active != PanelEditor {
			m.active = PanelSchema
			m.updatePanelActive()
			return m, nil
		}
	case "2":
		if m.active != PanelEditor {
			m.active = PanelEditor
			m.updatePanelActive()
			return m, nil
		}
	case "3":
		if m.active != PanelEditor {
			m.active = PanelResults
			m.updatePanelActive()
			return m, nil
		}
	case "y":
		if m.active == PanelResults && m.lastResult != nil {
			return m, m.copyCellCmd()
		}
	case "Y":
		if m.active == PanelResults && m.lastResult != nil {
			return m, m.copyRowCmd()
		}
	case "e":
		if m.active == PanelResults && m.lastResult != nil {
			m.exportDlg.Show()
			return m, nil
		}
	case "i":
		if m.active == PanelResults && m.lastResult != nil {
			// Start inline edit
			m.inlineEdit.StartEdit(m.lastResult, "", m.results.CursorRow(), 0, nil)
			return m, nil
		}
	case "enter":
		if m.active == PanelResults && m.lastResult != nil && len(m.lastResult.Rows) > 0 {
			m.rowDetail.Show(m.lastResult, m.results.CursorRow())
			return m, nil
		}
	case "ctrl+e":
		if m.active == PanelEditor {
			query := m.editor.GetQuery()
			if strings.TrimSpace(query) != "" {
				return m, m.explainQuery(query)
			}
		}
	case "ctrl+t":
		m.tabs.SetActiveQuery(m.editor.GetQuery())
		m.tabs.AddTab()
		m.editor.SetQuery("")
		return m, nil
	case "ctrl+w":
		if m.tabs.Count() > 1 {
			m.tabs.CloseTab()
			m.editor.SetQuery(m.tabs.GetActiveQuery())
		}
		return m, nil
	case "ctrl+b":
		// Begin transaction
		if !m.txIndicator.IsActive() {
			m.txIndicator.Begin()
			m.statusBar.SetMessage("Transaction started")
			return m, m.executeQuery("BEGIN")
		} else {
			// Show commit/rollback dialog
			m.txConfirm.Show("commit")
			return m, nil
		}
	case "ctrl+shift+b":
		if m.txIndicator.IsActive() {
			m.txConfirm.Show("rollback")
			return m, nil
		}
	case "ctrl+n":
		// Switch database connection
		if m.multiDB.Count() > 1 {
			m.multiDB.NextConnection()
			conn := m.multiDB.ActiveConnection()
			if conn != nil {
				m.driver = conn.Driver
				m.schema.SetDriver(m.driver)
				m.statusBar.SetConnection(conn.DBType, "", conn.Name)
				return m, ui.LoadSchemaCmd(m.driver)
			}
		}
	}

	// Panel-specific keybindings
	switch m.active {
	case PanelSchema:
		cmd, tableMsg := m.schema.Update(msg)
		if tableMsg != nil {
			return m, m.previewTable(tableMsg)
		}
		return m, cmd
	case PanelEditor:
		cmd := m.editor.Update(msg)
		return m, cmd
	case PanelResults:
		cmd := m.results.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleSchemaSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.searchMode = false
		m.fuzzyFilter.SetActive(false)
		return m, nil
	case tea.KeyEnter:
		m.searchMode = false
		m.fuzzyFilter.SetActive(false)
		return m, nil
	case tea.KeyBackspace:
		m.fuzzyFilter.Backspace()
		return m, nil
	case tea.KeyRunes:
		m.fuzzyFilter.AddChar(string(msg.Runes))
		return m, nil
	case tea.KeyUp:
		m.fuzzyFilter.MoveUp()
		return m, nil
	case tea.KeyDown:
		m.fuzzyFilter.MoveDown()
		return m, nil
	}
	return m, nil
}

func (m Model) handleTxConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		action := m.txConfirm.Action()
		m.txConfirm.Hide()
		if action == "commit" {
			m.txIndicator.Commit()
			m.statusBar.SetMessage("Transaction committed")
			return m, m.executeQuery("COMMIT")
		} else {
			m.txIndicator.Rollback()
			m.statusBar.SetMessage("Transaction rolled back")
			return m, m.executeQuery("ROLLBACK")
		}
	case "n", "N", "esc":
		m.txConfirm.Hide()
	}
	return m, nil
}

func (m *Model) handleCommandAction(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "execute_query":
		query := m.editor.GetQuery()
		if strings.TrimSpace(query) != "" {
			m.editor.AddToHistory(query)
			m.loading = true
			return m, m.executeQuery(query)
		}
	case "export_csv":
		if m.lastResult != nil {
			return m, m.doExport(ui.ExportRequestMsg{Format: ui.ExportCSV, FilePath: "export.csv"})
		}
	case "export_json":
		if m.lastResult != nil {
			return m, m.doExport(ui.ExportRequestMsg{Format: ui.ExportJSON, FilePath: "export.json"})
		}
	case "export_sql":
		if m.lastResult != nil {
			return m, m.doExport(ui.ExportRequestMsg{Format: ui.ExportSQL, FilePath: "export.sql"})
		}
	case "copy_cell":
		if m.lastResult != nil {
			return m, m.copyCellCmd()
		}
	case "copy_row":
		if m.lastResult != nil {
			return m, m.copyRowCmd()
		}
	case "row_detail":
		if m.lastResult != nil && len(m.lastResult.Rows) > 0 {
			m.rowDetail.Show(m.lastResult, m.results.CursorRow())
		}
	case "describe_table":
		node := m.schema.SelectedNode()
		if node != nil && node.Type == "table" {
			return m, m.executeQuery(fmt.Sprintf("SELECT * FROM %s LIMIT 0", node.Name))
		}
	case "refresh_schema":
		if m.driver != nil {
			return m, ui.LoadSchemaCmd(m.driver)
		}
	case "explain_query":
		query := m.editor.GetQuery()
		if strings.TrimSpace(query) != "" {
			return m, m.explainQuery(query)
		}
	case "clear_editor":
		m.editor.SetQuery("")
	case "toggle_help":
		m.help.Toggle()
	case "new_tab":
		m.tabs.SetActiveQuery(m.editor.GetQuery())
		m.tabs.AddTab()
		m.editor.SetQuery("")
	case "quit":
		m.multiDB.DisconnectAll()
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleTransaction(msg ui.TransactionMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case "begin":
		m.txIndicator.Begin()
		return m, m.executeQuery("BEGIN")
	case "commit":
		m.txIndicator.Commit()
		return m, m.executeQuery("COMMIT")
	case "rollback":
		m.txIndicator.Rollback()
		return m, m.executeQuery("ROLLBACK")
	}
	return m, nil
}

func (m *Model) cyclePanel(dir int) {
	m.active = Panel((int(m.active) + dir + 3) % 3)
}

func (m *Model) updatePanelActive() {
	m.schema.SetActive(m.active == PanelSchema)
	m.editor.SetActive(m.active == PanelEditor)
	m.results.SetActive(m.active == PanelResults)
}

func (m *Model) updateLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}

	schemaWidth := m.width / 4
	if schemaWidth < 20 {
		schemaWidth = 20
	}
	if schemaWidth > 40 {
		schemaWidth = 40
	}

	rightWidth := m.width - schemaWidth - 4
	editorHeight := (m.height - 3) / 2
	resultsHeight := m.height - 3 - editorHeight

	m.schema.SetSize(schemaWidth, m.height-3)
	m.editor.SetSize(rightWidth, editorHeight)
	m.results.SetSize(rightWidth, resultsHeight)
	m.statusBar.SetWidth(m.width)
	m.help.SetSize(m.width, m.height)
	m.cmdPalette.SetSize(m.width, m.height)
	m.rowDetail.SetSize(m.width, m.height)
	m.exportDlg.SetSize(m.width, m.height)
	m.txConfirm.SetSize(m.width, m.height)
	m.tabs.SetWidth(rightWidth)
	m.multiDB.SetWidth(m.width)
	m.inlineEdit.SetWidth(rightWidth)

	m.updatePanelActive()
}

func (m Model) executeQuery(query string) tea.Cmd {
	driver := m.driver
	return func() tea.Msg {
		result, err := driver.Execute(query)
		return queryResultMsg{result: result, err: err}
	}
}

func (m Model) explainQuery(query string) tea.Cmd {
	driver := m.driver
	return func() tea.Msg {
		explanation, err := driver.ExplainQuery(query)
		if err != nil {
			return queryResultMsg{err: err}
		}
		lines := strings.Split(explanation, "\n")
		var rows [][]string
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				rows = append(rows, []string{line})
			}
		}
		result := &db.QueryResult{
			Columns:  []string{"Query Plan"},
			Rows:     rows,
			RowCount: len(rows),
			Message:  "Query plan",
		}
		return queryResultMsg{result: result}
	}
}

func (m Model) previewTable(msg *ui.TableSelectedMsg) tea.Cmd {
	driver := m.driver
	return func() tea.Msg {
		result, err := driver.GetTablePreview(msg.Database, msg.Schema, msg.Table, 100)
		if err != nil {
			return queryResultMsg{err: err}
		}
		return queryResultMsg{result: result}
	}
}

func (m Model) doExport(req ui.ExportRequestMsg) tea.Cmd {
	result := m.lastResult
	return func() tea.Msg {
		if result == nil {
			return exportDoneMsg{err: fmt.Errorf("no results to export")}
		}
		f, err := os.Create(req.FilePath)
		if err != nil {
			return exportDoneMsg{err: err}
		}
		defer f.Close()

		switch req.Format {
		case ui.ExportCSV:
			err = export.CSV(f, result)
		case ui.ExportJSON:
			err = export.JSON(f, result)
		case ui.ExportSQL:
			err = export.SQL(f, "exported_data", result)
		}
		return exportDoneMsg{path: req.FilePath, err: err}
	}
}

func (m Model) copyCellCmd() tea.Cmd {
	result := m.lastResult
	row := m.results.CursorRow()
	return func() tea.Msg {
		if result == nil || row >= len(result.Rows) {
			return clipboardMsg{err: fmt.Errorf("no cell to copy")}
		}
		_, err := ui.CopyCellValue(result.Columns, result.Rows[row], 0)
		return clipboardMsg{err: err}
	}
}

func (m Model) copyRowCmd() tea.Cmd {
	result := m.lastResult
	row := m.results.CursorRow()
	return func() tea.Msg {
		if result == nil || row >= len(result.Rows) {
			return clipboardMsg{err: fmt.Errorf("no row to copy")}
		}
		_, err := ui.CopyRowAsJSON(result.Columns, result.Rows[row])
		return clipboardMsg{err: err}
	}
}

// View renders the entire application.
func (m Model) View() string {
	if !m.ready {
		return "Starting LazyDB..."
	}

	if m.err != nil && m.driver == nil {
		return fmt.Sprintf("Connection error: %v\n\nPress Ctrl+C to quit.", m.err)
	}

	// Overlays take priority (rendered on top)
	if m.txConfirm.IsVisible() {
		return m.txConfirm.View()
	}
	if m.cmdPalette.IsVisible() {
		return m.cmdPalette.View()
	}
	if m.rowDetail.IsVisible() {
		return m.rowDetail.View()
	}
	if m.exportDlg.IsVisible() {
		return m.exportDlg.View()
	}
	if m.help.IsVisible() {
		return m.help.View()
	}

	// Main layout
	dbBar := m.multiDB.View()

	schemaView := m.schema.View()

	tabBar := ""
	if m.tabs.Count() > 1 {
		tabBar = m.tabs.View() + "\n"
	}

	editorView := m.editor.View()
	resultsView := m.results.View()

	// Inline edit overlay
	if m.inlineEdit.IsActive() {
		resultsView += "\n" + m.inlineEdit.View()
	}

	rightPanel := tabBar + editorView + "\n" + resultsView
	mainContent := joinHorizontal(schemaView, rightPanel)

	// Status bar with transaction indicator
	txView := m.txIndicator.View()
	statusView := m.statusBar.View()

	// Loading indicator
	if m.loading {
		statusView = ui.StatusBarKeyStyle.Render(" LazyDB ") +
			ui.StatusBarStyle.Render(" Running query...")
	}

	return dbBar + mainContent + "\n" + txView + statusView
}

func joinHorizontal(left, right string) string {
	leftLines := splitLines(left)
	rightLines := splitLines(right)

	maxLen := len(leftLines)
	if len(rightLines) > maxLen {
		maxLen = len(rightLines)
	}

	var b strings.Builder
	for i := 0; i < maxLen; i++ {
		l := ""
		r := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		b.WriteString(l)
		b.WriteString(r)
		b.WriteByte('\n')
	}
	return b.String()
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
