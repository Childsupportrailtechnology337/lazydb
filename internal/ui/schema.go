package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
)

// TreeNode represents a node in the schema tree.
type TreeNode struct {
	Name     string
	Type     string // "database", "schema", "table", "column"
	Children []*TreeNode
	Expanded bool
	Depth    int
	Meta     string // Extra info like data type, row count
	Parent   *TreeNode
}

// SchemaPanel is the schema browser panel.
type SchemaPanel struct {
	nodes    []*TreeNode
	flat     []*TreeNode // Flattened visible nodes
	cursor   int
	width    int
	height   int
	filter   string
	active   bool
	driver   db.Driver
}

// NewSchemaPanel creates a new schema browser panel.
func NewSchemaPanel() SchemaPanel {
	return SchemaPanel{}
}

// SetDriver sets the database driver for loading schema data.
func (s *SchemaPanel) SetDriver(d db.Driver) {
	s.driver = d
}

// SetSize sets the panel dimensions.
func (s *SchemaPanel) SetSize(w, h int) {
	s.width = w
	s.height = h
}

// SetActive sets whether this panel is focused.
func (s *SchemaPanel) SetActive(active bool) {
	s.active = active
}

// flatten rebuilds the flat list of visible nodes.
func (s *SchemaPanel) flatten() {
	s.flat = nil
	var walk func(nodes []*TreeNode)
	walk = func(nodes []*TreeNode) {
		for _, n := range nodes {
			s.flat = append(s.flat, n)
			if n.Expanded && len(n.Children) > 0 {
				walk(n.Children)
			}
		}
	}
	walk(s.nodes)
}

// SelectedNode returns the currently selected node.
func (s *SchemaPanel) SelectedNode() *TreeNode {
	if len(s.flat) == 0 || s.cursor >= len(s.flat) {
		return nil
	}
	return s.flat[s.cursor]
}

// SchemaLoadedMsg is sent when schema data has been loaded.
type SchemaLoadedMsg struct {
	Nodes []*TreeNode
	Err   error
}

// TableSelectedMsg is sent when a table is selected for preview.
type TableSelectedMsg struct {
	Database string
	Schema   string
	Table    string
}

// LoadSchemaCmd loads the schema tree from the database.
func LoadSchemaCmd(d db.Driver) tea.Cmd {
	return func() tea.Msg {
		var nodes []*TreeNode

		databases, err := d.GetDatabases()
		if err != nil {
			// Single database mode (e.g., SQLite)
			schemas, err := d.GetSchemas("")
			if err != nil {
				return SchemaLoadedMsg{Err: err}
			}
			for _, schemaName := range schemas {
				schemaNode := &TreeNode{
					Name:  schemaName,
					Type:  "schema",
					Depth: 0,
				}
				tables, err := d.GetTables("", schemaName)
				if err == nil {
					for _, t := range tables {
						tableNode := &TreeNode{
							Name:   t.Name,
							Type:   "table",
							Depth:  1,
							Meta:   fmt.Sprintf("%d rows", t.RowCount),
							Parent: schemaNode,
						}
						cols, err := d.GetColumns("", schemaName, t.Name)
						if err == nil {
							for _, c := range cols {
								meta := c.DataType
								if c.PrimaryKey {
									meta += " PK"
								}
								if !c.Nullable {
									meta += " NOT NULL"
								}
								tableNode.Children = append(tableNode.Children, &TreeNode{
									Name:   c.Name,
									Type:   "column",
									Depth:  2,
									Meta:   meta,
									Parent: tableNode,
								})
							}
						}
						schemaNode.Children = append(schemaNode.Children, tableNode)
					}
				}
				nodes = append(nodes, schemaNode)
			}
			// Auto-expand first schema
			if len(nodes) > 0 {
				nodes[0].Expanded = true
			}
			return SchemaLoadedMsg{Nodes: nodes}
		}

		for _, dbName := range databases {
			dbNode := &TreeNode{
				Name:  dbName,
				Type:  "database",
				Depth: 0,
			}
			schemas, err := d.GetSchemas(dbName)
			if err == nil {
				for _, schemaName := range schemas {
					schemaNode := &TreeNode{
						Name:   schemaName,
						Type:   "schema",
						Depth:  1,
						Parent: dbNode,
					}
					tables, err := d.GetTables(dbName, schemaName)
					if err == nil {
						for _, t := range tables {
							tableNode := &TreeNode{
								Name:   t.Name,
								Type:   "table",
								Depth:  2,
								Meta:   fmt.Sprintf("%d rows", t.RowCount),
								Parent: schemaNode,
							}
							cols, err := d.GetColumns(dbName, schemaName, t.Name)
							if err == nil {
								for _, c := range cols {
									meta := c.DataType
									if c.PrimaryKey {
										meta += " PK"
									}
									tableNode.Children = append(tableNode.Children, &TreeNode{
										Name:   c.Name,
										Type:   "column",
										Depth:  3,
										Meta:   meta,
										Parent: tableNode,
									})
								}
							}
							schemaNode.Children = append(schemaNode.Children, tableNode)
						}
					}
					dbNode.Children = append(dbNode.Children, schemaNode)
				}
			}
			nodes = append(nodes, dbNode)
		}

		// Auto-expand first database
		if len(nodes) > 0 {
			nodes[0].Expanded = true
			if len(nodes[0].Children) > 0 {
				nodes[0].Children[0].Expanded = true
			}
		}

		return SchemaLoadedMsg{Nodes: nodes}
	}
}

// Update handles input for the schema panel.
func (s *SchemaPanel) Update(msg tea.KeyMsg) (tea.Cmd, *TableSelectedMsg) {
	switch msg.String() {
	case "j", "down":
		if s.cursor < len(s.flat)-1 {
			s.cursor++
		}
	case "k", "up":
		if s.cursor > 0 {
			s.cursor--
		}
	case "enter":
		node := s.SelectedNode()
		if node != nil && len(node.Children) > 0 {
			node.Expanded = !node.Expanded
			s.flatten()
		}
	case " ":
		// Preview table
		node := s.SelectedNode()
		if node != nil && node.Type == "table" {
			tbl := node.Name
			schema := ""
			database := ""
			if node.Parent != nil {
				schema = node.Parent.Name
				if node.Parent.Parent != nil {
					database = node.Parent.Parent.Name
				}
			}
			return nil, &TableSelectedMsg{Database: database, Schema: schema, Table: tbl}
		}
	case "g":
		s.cursor = 0
	case "G":
		if len(s.flat) > 0 {
			s.cursor = len(s.flat) - 1
		}
	}
	return nil, nil
}

// HandleSchemaLoaded processes the schema loaded message.
func (s *SchemaPanel) HandleSchemaLoaded(msg SchemaLoadedMsg) {
	if msg.Err == nil {
		s.nodes = msg.Nodes
		s.flatten()
	}
}

// View renders the schema panel.
func (s *SchemaPanel) View() string {
	var b strings.Builder

	title := PanelTitleStyle.Render("Schema")
	b.WriteString(title + "\n")

	if len(s.flat) == 0 {
		b.WriteString(MutedText.Render("  No schema loaded"))
		return s.wrapInBorder(b.String())
	}

	// Calculate visible range with scrolling
	visibleHeight := s.height - 3 // Account for title + border
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	start := 0
	if s.cursor >= visibleHeight {
		start = s.cursor - visibleHeight + 1
	}
	end := start + visibleHeight
	if end > len(s.flat) {
		end = len(s.flat)
	}

	for i := start; i < end; i++ {
		node := s.flat[i]
		indent := strings.Repeat("  ", node.Depth)

		var icon string
		switch node.Type {
		case "database":
			if node.Expanded {
				icon = "▼ "
			} else {
				icon = "▶ "
			}
		case "schema":
			if node.Expanded {
				icon = "▼ "
			} else {
				icon = "▶ "
			}
		case "table":
			if node.Expanded {
				icon = "▼ "
			} else {
				icon = "▶ "
			}
		case "column":
			icon = "  "
		}

		line := fmt.Sprintf("%s%s%s", indent, icon, node.Name)
		if node.Meta != "" {
			line += MutedText.Render(" " + node.Meta)
		}

		maxWidth := s.width - 4
		if maxWidth > 0 && lipgloss.Width(line) > maxWidth {
			line = line[:maxWidth]
		}

		if i == s.cursor && s.active {
			line = TreeNodeSelectedStyle.Render(line)
		} else {
			line = TreeNodeStyle.Render(line)
		}

		b.WriteString(line + "\n")
	}

	return s.wrapInBorder(b.String())
}

func (s *SchemaPanel) wrapInBorder(content string) string {
	style := InactivePanelBorder
	if s.active {
		style = ActivePanelBorder
	}
	return style.Width(s.width).Height(s.height).Render(content)
}
