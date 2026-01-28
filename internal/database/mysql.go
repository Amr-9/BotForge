package database

import (
	"context"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// ============================================
// Types & Constants
// ============================================

// MySQL wraps the sqlx.DB connection
type MySQL struct {
	db *sqlx.DB
}

// ============================================
// Constructor & Connection
// ============================================

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

// ============================================
// Schema Definitions
// ============================================

// baseTableQueries contains CREATE TABLE statements for all base tables
var baseTableQueries = []string{
	// Bots table
	`CREATE TABLE IF NOT EXISTS bots (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		token VARCHAR(255) NOT NULL UNIQUE,
		owner_chat_id BIGINT NOT NULL,
		is_active BOOLEAN DEFAULT TRUE,
		deleted_at TIMESTAMP NULL DEFAULT NULL,
		start_message TEXT,
		forward_auto_replies BOOLEAN DEFAULT TRUE,
		forced_sub_enabled BOOLEAN DEFAULT FALSE,
		forced_sub_message TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_owner (owner_chat_id),
		INDEX idx_active (is_active),
		INDEX idx_deleted (deleted_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

	// Message logs table
	`CREATE TABLE IF NOT EXISTS message_logs (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		admin_msg_id INT NOT NULL,
		user_chat_id BIGINT NOT NULL,
		bot_id BIGINT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_lookup (admin_msg_id, bot_id),
		FOREIGN KEY (bot_id) REFERENCES bots(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

	// Banned users table
	`CREATE TABLE IF NOT EXISTS banned_users (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		bot_id BIGINT NOT NULL,
		user_chat_id BIGINT NOT NULL,
		banned_by BIGINT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY uk_bot_user (bot_id, user_chat_id),
		INDEX idx_bot_id (bot_id),
		FOREIGN KEY (bot_id) REFERENCES bots(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

	// Auto replies table
	`CREATE TABLE IF NOT EXISTS auto_replies (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		bot_id BIGINT NOT NULL,
		trigger_word VARCHAR(255) NOT NULL,
		response TEXT NOT NULL,
		trigger_type ENUM('keyword', 'command') NOT NULL DEFAULT 'keyword',
		match_type ENUM('exact', 'contains') DEFAULT 'contains',
		is_active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY idx_bot_trigger (bot_id, trigger_word, trigger_type),
		INDEX idx_auto_replies_bot (bot_id, is_active),
		FOREIGN KEY (bot_id) REFERENCES bots(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

	// Scheduled messages table
	`CREATE TABLE IF NOT EXISTS scheduled_messages (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		bot_id BIGINT NOT NULL,
		owner_chat_id BIGINT NOT NULL,
		message_type ENUM('text', 'photo', 'video', 'document') NOT NULL DEFAULT 'text',
		message_text TEXT,
		file_id VARCHAR(255),
		caption TEXT,
		schedule_type ENUM('once', 'daily', 'weekly') NOT NULL,
		scheduled_time DATETIME NOT NULL,
		time_of_day TIME,
		day_of_week TINYINT,
		status ENUM('pending', 'sent', 'failed', 'paused', 'cancelled') NOT NULL DEFAULT 'pending',
		last_sent_at DATETIME NULL,
		next_run_at DATETIME NULL,
		failure_reason TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		INDEX idx_bot_id (bot_id),
		INDEX idx_status_next_run (status, next_run_at),
		INDEX idx_owner (owner_chat_id),
		FOREIGN KEY (bot_id) REFERENCES bots(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

	// Forced channels table (for forced subscription feature)
	`CREATE TABLE IF NOT EXISTS forced_channels (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		bot_id BIGINT NOT NULL,
		channel_id BIGINT NOT NULL,
		channel_username VARCHAR(255),
		channel_title VARCHAR(255),
		invite_link VARCHAR(255),
		is_active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY uk_bot_channel (bot_id, channel_id),
		INDEX idx_bot_active (bot_id, is_active),
		FOREIGN KEY (bot_id) REFERENCES bots(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
}

// ============================================
// Migration Functions
// ============================================

// migrate runs all database migrations
func (m *MySQL) migrate() error {
	for _, query := range baseTableQueries {
		if _, err := m.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}
	return nil
}

// ============================================
// Helper Functions
// ============================================

// addColumnIfNotExists safely adds a column if it doesn't exist
func (m *MySQL) addColumnIfNotExists(table, column, definition string) error {
	var count int
	query := `SELECT COUNT(*) FROM information_schema.COLUMNS
			  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`
	if err := m.db.Get(&count, query, table, column); err != nil {
		return fmt.Errorf("failed to check column existence: %w", err)
	}

	if count == 0 {
		alterQuery := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)
		if _, err := m.db.Exec(alterQuery); err != nil {
			return fmt.Errorf("failed to add column %s: %w", column, err)
		}
		log.Printf("Added column %s to table %s", column, table)
	}

	return nil
}
