package demo

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// CreateDemoDatabase creates a temporary SQLite database populated with
// realistic sample data and returns the file path to the database.
func CreateDemoDatabase() (string, error) {
	tmpDir, err := os.MkdirTemp("", "lazydb-demo-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	dbPath := filepath.Join(tmpDir, "demo.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Enable foreign keys and WAL mode.
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return "", err
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return "", err
	}

	if err := createSchema(db); err != nil {
		return "", fmt.Errorf("failed to create schema: %w", err)
	}

	rng := rand.New(rand.NewSource(42))

	if err := populateCategories(db); err != nil {
		return "", fmt.Errorf("failed to populate categories: %w", err)
	}
	if err := populateProducts(db, rng); err != nil {
		return "", fmt.Errorf("failed to populate products: %w", err)
	}
	if err := populateUsers(db, rng); err != nil {
		return "", fmt.Errorf("failed to populate users: %w", err)
	}
	if err := populateOrders(db, rng); err != nil {
		return "", fmt.Errorf("failed to populate orders: %w", err)
	}
	if err := populateOrderItems(db, rng); err != nil {
		return "", fmt.Errorf("failed to populate order_items: %w", err)
	}
	if err := populateReviews(db, rng); err != nil {
		return "", fmt.Errorf("failed to populate reviews: %w", err)
	}

	return dbPath, nil
}

func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE categories (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT    NOT NULL UNIQUE,
		description TEXT,
		created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE products (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT    NOT NULL,
		description TEXT,
		price       REAL    NOT NULL,
		category_id INTEGER NOT NULL,
		stock       INTEGER NOT NULL DEFAULT 0,
		weight_kg   REAL,
		is_active   INTEGER NOT NULL DEFAULT 1,
		created_at  TEXT    NOT NULL,
		updated_at  TEXT    NOT NULL,
		FOREIGN KEY (category_id) REFERENCES categories(id)
	);

	CREATE INDEX idx_products_category ON products(category_id);
	CREATE INDEX idx_products_price    ON products(price);
	CREATE INDEX idx_products_active   ON products(is_active);

	CREATE TABLE users (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		first_name   TEXT    NOT NULL,
		last_name    TEXT    NOT NULL,
		email        TEXT    NOT NULL UNIQUE,
		city         TEXT,
		state        TEXT,
		address      TEXT,
		phone        TEXT,
		is_active    INTEGER NOT NULL DEFAULT 1,
		credit_limit REAL    DEFAULT 0.0,
		created_at   TEXT    NOT NULL,
		last_login   TEXT
	);

	CREATE INDEX idx_users_email ON users(email);
	CREATE INDEX idx_users_city  ON users(city);

	CREATE TABLE orders (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id     INTEGER NOT NULL,
		status      TEXT    NOT NULL DEFAULT 'pending',
		total       REAL    NOT NULL DEFAULT 0.0,
		tax         REAL    NOT NULL DEFAULT 0.0,
		shipping    REAL    NOT NULL DEFAULT 0.0,
		notes       TEXT,
		ordered_at  TEXT    NOT NULL,
		shipped_at  TEXT,
		delivered_at TEXT,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE INDEX idx_orders_user   ON orders(user_id);
	CREATE INDEX idx_orders_status ON orders(status);
	CREATE INDEX idx_orders_date   ON orders(ordered_at);

	CREATE TABLE order_items (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		order_id   INTEGER NOT NULL,
		product_id INTEGER NOT NULL,
		quantity   INTEGER NOT NULL DEFAULT 1,
		unit_price REAL    NOT NULL,
		discount   REAL    NOT NULL DEFAULT 0.0,
		FOREIGN KEY (order_id)   REFERENCES orders(id),
		FOREIGN KEY (product_id) REFERENCES products(id)
	);

	CREATE INDEX idx_order_items_order   ON order_items(order_id);
	CREATE INDEX idx_order_items_product ON order_items(product_id);

	CREATE TABLE reviews (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id    INTEGER NOT NULL,
		product_id INTEGER NOT NULL,
		rating     INTEGER NOT NULL CHECK(rating >= 1 AND rating <= 5),
		title      TEXT,
		body       TEXT,
		is_verified INTEGER NOT NULL DEFAULT 0,
		created_at TEXT    NOT NULL,
		FOREIGN KEY (user_id)    REFERENCES users(id),
		FOREIGN KEY (product_id) REFERENCES products(id)
	);

	CREATE INDEX idx_reviews_product ON reviews(product_id);
	CREATE INDEX idx_reviews_user    ON reviews(user_id);
	CREATE INDEX idx_reviews_rating  ON reviews(rating);
	`
	_, err := db.Exec(schema)
	return err
}

func populateCategories(db *sql.DB) error {
	descriptions := []string{
		"Gadgets, devices, and electronic accessories",
		"Apparel, footwear, and fashion accessories",
		"Cookware, utensils, and home essentials",
		"Fiction, non-fiction, and technical books",
		"Fitness gear, camping, and outdoor equipment",
		"Pens, notebooks, and desk accessories",
		"Skincare, wellness, and personal care",
		"Board games, puzzles, and building sets",
		"Car parts, tools, and vehicle accessories",
		"Planters, furniture, and outdoor decor",
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO categories (name, description) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, cat := range Categories {
		if _, err := stmt.Exec(cat, descriptions[i]); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// categoryIDMap maps category name to its database ID.
func categoryIDMap(db *sql.DB) (map[string]int, error) {
	rows, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]int)
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		m[name] = id
	}
	return m, rows.Err()
}

func populateProducts(db *sql.DB, rng *rand.Rand) error {
	catMap, err := categoryIDMap(db)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO products
		(name, description, price, category_id, stock, weight_kg, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, p := range Products {
		catID := catMap[p.Category]
		price := roundTo(5.0+rng.Float64()*295.0, 2) // $5 - $300
		stock := rng.Intn(500)
		weight := roundTo(0.1+rng.Float64()*15.0, 1)
		isActive := 1
		if rng.Float64() < 0.1 {
			isActive = 0
		}
		created := baseDate.Add(time.Duration(rng.Intn(365*24)) * time.Hour)
		updated := created.Add(time.Duration(rng.Intn(60*24)) * time.Hour)
		desc := fmt.Sprintf("High quality %s. Perfect for everyday use.", strings.ToLower(p.Name))

		if _, err := stmt.Exec(p.Name, desc, price, catID, stock, weight, isActive,
			created.Format(time.DateTime), updated.Format(time.DateTime)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func populateUsers(db *sql.DB, rng *rand.Rand) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO users
		(first_name, last_name, email, city, state, address, phone, is_active, credit_limit, created_at, last_login)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	usedEmails := make(map[string]bool)
	baseDate := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 100; i++ {
		first := FirstNames[rng.Intn(len(FirstNames))]
		last := LastNames[rng.Intn(len(LastNames))]

		// Generate a unique email.
		var email string
		for {
			domain := EmailDomains[rng.Intn(len(EmailDomains))]
			suffix := ""
			if rng.Float64() < 0.5 {
				suffix = fmt.Sprintf("%d", rng.Intn(999))
			}
			email = fmt.Sprintf("%s.%s%s@%s",
				strings.ToLower(first), strings.ToLower(last), suffix, domain)
			if !usedEmails[email] {
				usedEmails[email] = true
				break
			}
		}

		cityIdx := rng.Intn(len(Cities))
		city := Cities[cityIdx]
		state := States[cityIdx]
		streetNum := rng.Intn(9999) + 1
		street := StreetNames[rng.Intn(len(StreetNames))]
		address := fmt.Sprintf("%d %s", streetNum, street)
		phone := fmt.Sprintf("(%03d) %03d-%04d", rng.Intn(900)+100, rng.Intn(900)+100, rng.Intn(10000))

		isActive := 1
		if rng.Float64() < 0.08 {
			isActive = 0
		}
		creditLimit := roundTo(rng.Float64()*10000.0, 2)
		createdAt := baseDate.Add(time.Duration(rng.Intn(500*24)) * time.Hour)

		var lastLogin *string
		if rng.Float64() < 0.85 {
			ll := createdAt.Add(time.Duration(rng.Intn(200*24)) * time.Hour).Format(time.DateTime)
			lastLogin = &ll
		}

		if _, err := stmt.Exec(first, last, email, city, state, address, phone,
			isActive, creditLimit, createdAt.Format(time.DateTime), lastLogin); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func populateOrders(db *sql.DB, rng *rand.Rand) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO orders
		(user_id, status, total, tax, shipping, notes, ordered_at, shipped_at, delivered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	orderNotes := []string{
		"", "", "", "", "", // Most orders have no notes.
		"Please leave at front door",
		"Gift wrap requested",
		"Fragile items - handle with care",
		"Call before delivery",
		"Leave with neighbor if not home",
	}

	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 500; i++ {
		userID := rng.Intn(100) + 1
		status := OrderStatuses[rng.Intn(len(OrderStatuses))]
		total := roundTo(10.0+rng.Float64()*490.0, 2)
		tax := roundTo(total*0.08, 2)
		shipping := 0.0
		if total < 50.0 {
			shipping = roundTo(5.0+rng.Float64()*10.0, 2)
		}
		notes := orderNotes[rng.Intn(len(orderNotes))]
		orderedAt := baseDate.Add(time.Duration(rng.Intn(365*24)) * time.Hour)

		var shippedAt, deliveredAt *string
		if status == "shipped" || status == "delivered" {
			s := orderedAt.Add(time.Duration(rng.Intn(72)+24) * time.Hour).Format(time.DateTime)
			shippedAt = &s
		}
		if status == "delivered" {
			d := orderedAt.Add(time.Duration(rng.Intn(120)+96) * time.Hour).Format(time.DateTime)
			deliveredAt = &d
		}

		var notesVal *string
		if notes != "" {
			notesVal = &notes
		}

		if _, err := stmt.Exec(userID, status, total, tax, shipping, notesVal,
			orderedAt.Format(time.DateTime), shippedAt, deliveredAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func populateOrderItems(db *sql.DB, rng *rand.Rand) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO order_items
		(order_id, product_id, quantity, unit_price, discount)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	productCount := len(Products)

	for i := 0; i < 1000; i++ {
		orderID := rng.Intn(500) + 1
		productID := rng.Intn(productCount) + 1
		quantity := rng.Intn(5) + 1
		unitPrice := roundTo(5.0+rng.Float64()*295.0, 2)
		discount := 0.0
		if rng.Float64() < 0.2 {
			discount = roundTo(rng.Float64()*unitPrice*0.3, 2)
		}

		if _, err := stmt.Exec(orderID, productID, quantity, unitPrice, discount); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func populateReviews(db *sql.DB, rng *rand.Rand) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO reviews
		(user_id, product_id, rating, title, body, is_verified, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	productCount := len(Products)
	reviewTitles := []string{
		"Great product!", "Worth every penny", "Not bad", "Could be better",
		"Amazing quality", "Decent purchase", "Love it!", "Disappointing",
		"Exactly as described", "Exceeded expectations", "Just okay",
		"Highly recommend", "Will buy again", "Perfect gift", "Good value",
	}

	baseDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 200; i++ {
		userID := rng.Intn(100) + 1
		productID := rng.Intn(productCount) + 1
		// Weight ratings toward higher values (more realistic).
		rating := rng.Intn(5) + 1
		if rng.Float64() < 0.4 {
			rating = 4 + rng.Intn(2) // Push 40% to 4-5 stars
		}
		title := reviewTitles[rng.Intn(len(reviewTitles))]
		body := ReviewTemplates[rng.Intn(len(ReviewTemplates))]
		isVerified := 0
		if rng.Float64() < 0.65 {
			isVerified = 1
		}
		createdAt := baseDate.Add(time.Duration(rng.Intn(300*24)) * time.Hour)

		if _, err := stmt.Exec(userID, productID, rating, title, body, isVerified,
			createdAt.Format(time.DateTime)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func roundTo(val float64, decimals int) float64 {
	p := 1.0
	for i := 0; i < decimals; i++ {
		p *= 10
	}
	return float64(int(val*p+0.5)) / p
}
