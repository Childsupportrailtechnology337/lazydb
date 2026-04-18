package cmd

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/aymenhmaidiwastaken/lazydb/internal/app"
	"github.com/aymenhmaidiwastaken/lazydb/internal/config"
	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
	"github.com/aymenhmaidiwastaken/lazydb/internal/demo"
	"github.com/aymenhmaidiwastaken/lazydb/internal/ui"
)

var (
	version   = "dev"
	commit    = "none"
	date      = "unknown"
	themeName string
)

var rootCmd = &cobra.Command{
	Use:   "lazydb [connection-string | profile-name]",
	Short: "LazyDB — Universal Database TUI",
	Long: `One TUI to query them all.
Connect to PostgreSQL, MySQL, SQLite, MongoDB, and Redis with a unified interface.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Apply theme
		if themeName != "" {
			ui.ApplyTheme(ui.GetTheme(themeName))
		}

		// Check for project config
		projCfg, _, _ := config.LoadProjectConfig()

		if len(args) == 0 {
			// If project config has a default connection, use it
			if projCfg != nil && projCfg.DefaultConn != "" {
				for _, c := range projCfg.Connections {
					if c.Name == projCfg.DefaultConn {
						return runTUI(c)
					}
				}
			}
			return fmt.Errorf("please provide a connection string, file path, or profile name\n\nExamples:\n  lazydb demo\n  lazydb ./mydb.sqlite\n  lazydb postgres://user:pass@localhost/mydb\n  lazydb mysql://user:pass@localhost/mydb\n  lazydb mongodb://localhost:27017/mydb\n  lazydb redis://localhost:6379\n  lazydb my-profile\n\nThemes: --theme catppuccin-mocha|dracula|tokyo-night|gruvbox|nord|one-dark|rose-pine")
		}

		connArg := args[0]
		var connConfig db.ConnectionConfig

		// Check project config first
		if projCfg != nil {
			for _, c := range projCfg.Connections {
				if c.Name == connArg {
					connConfig = c
					break
				}
			}
			// Apply project theme if set
			if projCfg.Theme != "" && themeName == "" {
				ui.ApplyTheme(ui.GetTheme(projCfg.Theme))
			}
		}

		// Check saved profiles
		if connConfig.Type == "" {
			cfg, err := config.Load()
			if err == nil && cfg != nil {
				for _, c := range cfg.Connections {
					if c.Name == connArg {
						connConfig = c
						break
					}
				}
			}
		}

		// Parse as connection string
		if connConfig.Type == "" {
			var err error
			connConfig, err = db.ParseConnectionString(connArg)
			if err != nil {
				return fmt.Errorf("invalid connection string: %w", err)
			}
		}

		return runTUI(connConfig)
	},
}

func runTUI(connConfig db.ConnectionConfig) error {
	var model tea.Model
	if themeName != "" {
		model = app.NewWithTheme(connConfig, themeName)
	} else {
		model = app.New(connConfig)
	}

	p := tea.NewProgram(model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running LazyDB: %w", err)
	}
	return nil
}

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Launch with a demo SQLite database",
	Long:  "Start LazyDB with a pre-populated SQLite database to explore features without any setup.",
	RunE: func(cmd *cobra.Command, args []string) error {
		demoPath, err := demo.CreateDemoDatabase()
		if err != nil {
			return fmt.Errorf("failed to create demo database: %w", err)
		}
		defer os.Remove(demoPath)

		connConfig := db.ConnectionConfig{
			Type:     "sqlite",
			FilePath: demoPath,
			Name:     "Demo Database",
		}

		if themeName != "" {
			ui.ApplyTheme(ui.GetTheme(themeName))
		}

		return runTUI(connConfig)
	},
}

func createQuickDemo() (string, error) {
	f, err := os.CreateTemp("", "lazydb-demo-*.db")
	if err != nil {
		return "", err
	}
	path := f.Name()
	f.Close()

	driver := db.NewSQLiteDriver()
	if err := driver.Connect(db.ConnectionConfig{FilePath: path}); err != nil {
		return "", err
	}
	defer driver.Disconnect()

	// Create tables and data
	stmts := []string{
		`CREATE TABLE categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			city TEXT,
			active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			category_id INTEGER REFERENCES categories(id),
			price REAL NOT NULL,
			stock INTEGER DEFAULT 0,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER REFERENCES users(id),
			status TEXT DEFAULT 'pending',
			total REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			shipped_at DATETIME
		)`,
		`CREATE TABLE order_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER REFERENCES orders(id),
			product_id INTEGER REFERENCES products(id),
			quantity INTEGER NOT NULL,
			unit_price REAL NOT NULL
		)`,
		`CREATE TABLE reviews (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER REFERENCES users(id),
			product_id INTEGER REFERENCES products(id),
			rating INTEGER CHECK(rating BETWEEN 1 AND 5),
			comment TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Indexes
		`CREATE INDEX idx_users_email ON users(email)`,
		`CREATE INDEX idx_users_city ON users(city)`,
		`CREATE INDEX idx_orders_user ON orders(user_id)`,
		`CREATE INDEX idx_orders_status ON orders(status)`,
		`CREATE INDEX idx_order_items_order ON order_items(order_id)`,
		`CREATE INDEX idx_reviews_product ON reviews(product_id)`,
	}

	for _, stmt := range stmts {
		if _, err := driver.Execute(stmt); err != nil {
			return "", fmt.Errorf("schema creation failed: %w", err)
		}
	}

	// Insert categories
	categories := []string{"Electronics", "Books", "Clothing", "Home & Garden", "Sports", "Toys", "Food", "Health", "Automotive", "Music"}
	for _, cat := range categories {
		driver.Execute(fmt.Sprintf("INSERT INTO categories (name, description) VALUES ('%s', 'Category for %s products')", cat, strings.ToLower(cat)))
	}

	// Insert users
	firstNames := []string{"Alice", "Bob", "Carol", "Dave", "Eve", "Frank", "Grace", "Hank", "Iris", "Jack",
		"Kate", "Leo", "Mia", "Nick", "Olivia", "Paul", "Quinn", "Rose", "Sam", "Tina",
		"Uma", "Vic", "Wendy", "Xander", "Yara", "Zane", "Amy", "Ben", "Chloe", "Dan",
		"Emma", "Finn", "Gina", "Hugo", "Ivy", "Jake", "Lily", "Max", "Nora", "Oscar",
		"Pia", "Reed", "Sara", "Tom", "Una", "Val", "Will", "Xia", "Yuri", "Zoe"}
	lastNames := []string{"Smith", "Jones", "Brown", "Wilson", "Taylor", "Davis", "Clark", "Lewis", "Hall", "Young",
		"King", "Wright", "Green", "Baker", "Adams", "Nelson", "Carter", "Mitchell", "Roberts", "Turner"}
	cities := []string{"New York", "London", "Tokyo", "Paris", "Berlin", "Sydney", "Toronto", "Mumbai", "Dubai", "Singapore",
		"Amsterdam", "Barcelona", "Seoul", "Istanbul", "Bangkok", "Rome", "Vienna", "Prague", "Dublin", "Oslo"}

	for i := 0; i < 80; i++ {
		fn := firstNames[i%len(firstNames)]
		ln := lastNames[i%len(lastNames)]
		city := cities[i%len(cities)]
		email := strings.ToLower(fn) + "." + strings.ToLower(ln) + fmt.Sprintf("%d", i) + "@example.com"
		active := 1
		if i%7 == 0 {
			active = 0
		}
		driver.Execute(fmt.Sprintf(
			"INSERT INTO users (name, email, city, active, created_at) VALUES ('%s %s', '%s', '%s', %d, datetime('now', '-%d days'))",
			fn, ln, email, city, active, i*3))
	}

	// Insert products
	productNames := []string{
		"Wireless Mouse", "Mechanical Keyboard", "USB-C Hub", "Monitor Stand", "Webcam HD",
		"Noise Cancelling Headphones", "Laptop Sleeve", "Phone Charger", "Smart Watch", "Bluetooth Speaker",
		"LED Desk Lamp", "Ergonomic Chair", "Standing Desk", "Cable Organizer", "Mouse Pad XL",
		"External SSD 1TB", "Portable Battery", "Screen Protector", "Tablet Case", "HDMI Cable",
		"WiFi Router", "Ethernet Cable", "USB Flash Drive", "Memory Card", "Action Camera",
		"Drone Mini", "VR Headset", "Game Controller", "Streaming Mic", "Ring Light",
		"Fiction Novel", "Sci-Fi Anthology", "Cooking Guide", "History Book", "Art Collection",
		"Running Shoes", "Yoga Mat", "Water Bottle", "Gym Bag", "Tennis Racket",
		"Coffee Maker", "Blender", "Air Purifier", "Plant Pot", "Throw Blanket",
		"Vitamin D3", "Protein Powder", "Face Cream", "Sunscreen SPF50", "Hand Sanitizer"}

	for i, name := range productNames {
		catID := (i % 10) + 1
		price := float64(5+i*3) + 0.99
		stock := 10 + i*5
		driver.Execute(fmt.Sprintf(
			"INSERT INTO products (name, category_id, price, stock, description, created_at) VALUES ('%s', %d, %.2f, %d, 'High quality %s', datetime('now', '-%d days'))",
			name, catID, price, stock, strings.ToLower(name), i*2))
	}

	// Insert orders
	statuses := []string{"pending", "confirmed", "shipped", "delivered", "cancelled"}
	for i := 0; i < 300; i++ {
		userID := (i % 80) + 1
		status := statuses[i%len(statuses)]
		total := float64(15+i*2) + 0.50
		shipped := "NULL"
		if status == "shipped" || status == "delivered" {
			shipped = fmt.Sprintf("datetime('now', '-%d days')", i)
		}
		driver.Execute(fmt.Sprintf(
			"INSERT INTO orders (user_id, status, total, created_at, shipped_at) VALUES (%d, '%s', %.2f, datetime('now', '-%d days'), %s)",
			userID, status, total, i*2, shipped))
	}

	// Insert order items
	for i := 0; i < 800; i++ {
		orderID := (i % 300) + 1
		productID := (i % 50) + 1
		qty := 1 + i%5
		price := float64(5+(productID*3)) + 0.99
		driver.Execute(fmt.Sprintf(
			"INSERT INTO order_items (order_id, product_id, quantity, unit_price) VALUES (%d, %d, %d, %.2f)",
			orderID, productID, qty, price))
	}

	// Insert reviews
	comments := []string{
		"Great product, highly recommended!",
		"Works as expected, good value for money.",
		"Decent quality but could be better.",
		"Not what I expected, returning it.",
		"Absolutely love it! Will buy again.",
		"Fast shipping, great packaging.",
		"Average product, nothing special.",
		"Excellent build quality and design.",
		"Broke after a week, very disappointed.",
		"Perfect for my needs, 5 stars!",
	}
	for i := 0; i < 200; i++ {
		userID := (i % 80) + 1
		productID := (i % 50) + 1
		rating := (i % 5) + 1
		comment := comments[i%len(comments)]
		driver.Execute(fmt.Sprintf(
			"INSERT INTO reviews (user_id, product_id, rating, comment, created_at) VALUES (%d, %d, %d, '%s', datetime('now', '-%d days'))",
			userID, productID, rating, comment, i*3))
	}

	return path, nil
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("LazyDB %s\n", version)
		if commit != "none" {
			fmt.Printf("  commit: %s\n  built:  %s\n", commit, date)
		}
	},
}

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "List saved connection profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}

		if len(cfg.Connections) == 0 {
			fmt.Println("No saved profiles. Add connections to ~/.config/lazydb/connections.yaml")
			return nil
		}

		fmt.Println("Saved connection profiles:")
		for _, c := range cfg.Connections {
			dbType := c.Type
			target := c.Database
			if c.FilePath != "" {
				target = c.FilePath
			}
			if c.Host != "" {
				target = fmt.Sprintf("%s@%s:%d/%s", c.User, c.Host, c.Port, c.Database)
			}
			fmt.Printf("  %-20s %-10s %s\n", c.Name, dbType, target)
		}
		return nil
	},
}

var runCmd = &cobra.Command{
	Use:   "run [file.sql]",
	Short: "Execute a SQL file non-interactively",
	Long:  "Connect to the database and execute a SQL file, printing results to stdout.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		connStr, _ := cmd.Flags().GetString("connection")
		if connStr == "" {
			// Check project config
			projCfg, _, _ := config.LoadProjectConfig()
			if projCfg != nil && projCfg.DefaultConn != "" {
				for _, c := range projCfg.Connections {
					if c.Name == projCfg.DefaultConn {
						return runSQLFile(c, args[0])
					}
				}
			}
			return fmt.Errorf("no connection specified. Use --connection flag or .lazydb.yaml")
		}

		connConfig, err := db.ParseConnectionString(connStr)
		if err != nil {
			return err
		}
		return runSQLFile(connConfig, args[0])
	},
}

func runSQLFile(connConfig db.ConnectionConfig, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	driver, err := db.NewDriver(connConfig.Type)
	if err != nil {
		return err
	}
	if err := driver.Connect(connConfig); err != nil {
		return err
	}
	defer driver.Disconnect()

	sql := string(data)
	statements := splitStatements(sql)

	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		result, err := driver.Execute(stmt)
		if err != nil {
			return fmt.Errorf("statement %d failed: %w\n  %s", i+1, err, truncate(stmt, 100))
		}

		if result.Message != "" {
			fmt.Printf("-- Statement %d: %s\n", i+1, result.Message)
		}
		if len(result.Rows) > 0 {
			// Print as table
			fmt.Println(strings.Join(result.Columns, "\t"))
			fmt.Println(strings.Repeat("-", 40))
			for _, row := range result.Rows {
				fmt.Println(strings.Join(row, "\t"))
			}
			fmt.Printf("(%d rows, %s)\n\n", result.RowCount, result.Duration)
		}
	}

	return nil
}

func splitStatements(sql string) []string {
	var stmts []string
	var current strings.Builder
	inString := false
	stringChar := byte(0)

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		if inString {
			current.WriteByte(ch)
			if ch == stringChar {
				// Check for escaped quote
				if i+1 < len(sql) && sql[i+1] == stringChar {
					current.WriteByte(sql[i+1])
					i++
				} else {
					inString = false
				}
			}
			continue
		}

		if ch == '\'' || ch == '"' {
			inString = true
			stringChar = ch
			current.WriteByte(ch)
			continue
		}

		if ch == '-' && i+1 < len(sql) && sql[i+1] == '-' {
			// Line comment — skip to end of line
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			continue
		}

		if ch == ';' {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				stmts = append(stmts, stmt)
			}
			current.Reset()
			continue
		}

		current.WriteByte(ch)
	}

	// Last statement without semicolon
	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		stmts = append(stmts, stmt)
	}

	return stmts
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

var themesCmd = &cobra.Command{
	Use:   "themes",
	Short: "List available themes",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available themes:")
		for name := range ui.BuiltinThemes {
			fmt.Printf("  %s\n", name)
		}
		fmt.Println("\nUsage: lazydb --theme <name> <connection>")
		fmt.Println("Or set in .lazydb.yaml: theme: <name>")
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a .lazydb.yaml template in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := ".lazydb.yaml"
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf(".lazydb.yaml already exists")
		}
		if err := os.WriteFile(path, []byte(config.ExampleProjectConfig()), 0o644); err != nil {
			return err
		}
		fmt.Println("Created .lazydb.yaml — edit it with your database connections.")
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&themeName, "theme", "", "color theme (catppuccin-mocha, dracula, tokyo-night, gruvbox, nord, one-dark, rose-pine)")
	runCmd.Flags().StringP("connection", "c", "", "connection string for SQL execution")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(profilesCmd)
	rootCmd.AddCommand(demoCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(themesCmd)
	rootCmd.AddCommand(initCmd)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
