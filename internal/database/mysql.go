package database

import (
	"context"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// MySQL wraps the sqlx.DB connection
type MySQL struct {
	db *sqlx.DB
}

// NewMySQL creates a new MySQL connection with retry logic
func NewMySQL(dsn string) (*MySQL, error) {
	var db *sqlx.DB
	var err error

	// Retry connection with exponential backoff
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		db, err = sqlx.Connect("mysql", dsn)
		if err == nil {
			break
		}

		waitTime := time.Duration(1<<uint(i)) * time.Second
		log.Printf("Failed to connect to MySQL (attempt %d/%d): %v. Retrying in %v...",
			i+1, maxRetries, err, waitTime)
		time.Sleep(waitTime)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL after %d attempts: %w", maxRetries, err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	mysql := &MySQL{db: db}

	// Run migrations
	if err := mysql.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Connected to MySQL successfully")
	return mysql, nil
}

// migrate creates the required tables
func (m *MySQL) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS bots (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			token VARCHAR(255) NOT NULL UNIQUE,
			owner_chat_id BIGINT NOT NULL,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_owner (owner_chat_id),
			INDEX idx_active (is_active)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS message_logs (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			admin_msg_id INT NOT NULL,
			user_chat_id BIGINT NOT NULL,
			bot_id BIGINT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_lookup (admin_msg_id, bot_id),
			FOREIGN KEY (bot_id) REFERENCES bots(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
	}

	for _, query := range queries {
		if _, err := m.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

// DB returns the underlying sqlx.DB for advanced operations
func (m *MySQL) DB() *sqlx.DB {
	return m.db
}

// Close closes the database connection
func (m *MySQL) Close() error {
	return m.db.Close()
}

// Ping checks if database connection is alive
func (m *MySQL) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}
