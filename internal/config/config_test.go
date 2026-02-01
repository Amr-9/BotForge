package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Amr-9/botforge/internal/config"
)

// Helper function to clear all environment variables used by config
func clearEnv() {
	envVars := []string{
		"FACTORY_BOT_TOKEN",
		"ADMIN_ID",
		"WEBHOOK_URL",
		"PORT",
		"DB_HOST",
		"DB_USER",
		"DB_PASS",
		"DB_NAME",
		"REDIS_ADDR",
		"REDIS_PASSWORD",
		"REDIS_DB",
		"MESSAGE_TTL",
		"BOT_ENCRYPTION_KEY",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}

// Helper function to set minimal valid environment
func setValidEnv() {
	os.Setenv("FACTORY_BOT_TOKEN", "test-token-123")
	os.Setenv("WEBHOOK_URL", "https://example.com/webhook")
	os.Setenv("DB_HOST", "localhost:3306")
	os.Setenv("DB_USER", "root")
	os.Setenv("DB_PASS", "password")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("REDIS_ADDR", "localhost:6379")
	os.Setenv("BOT_ENCRYPTION_KEY", "12345678901234567890123456789012") // 32 chars
}

// ==================== Load Function Tests ====================

func TestLoad_ValidConfig(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.FactoryBotToken != "test-token-123" {
		t.Errorf("Expected token 'test-token-123', got '%s'", cfg.FactoryBotToken)
	}
	if cfg.WebhookURL != "https://example.com/webhook" {
		t.Errorf("Expected webhook URL 'https://example.com/webhook', got '%s'", cfg.WebhookURL)
	}
	if cfg.DBHost != "localhost:3306" {
		t.Errorf("Expected DB host 'localhost:3306', got '%s'", cfg.DBHost)
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Default PORT should be 4210
	if cfg.ServerPort != "4210" {
		t.Errorf("Expected default port '4210', got '%s'", cfg.ServerPort)
	}

	// Default REDIS_DB should be 0
	if cfg.RedisDB != 0 {
		t.Errorf("Expected default Redis DB 0, got %d", cfg.RedisDB)
	}

	// Default MESSAGE_TTL should be 48 hours
	if cfg.MessageTTL.Hours() != 48 {
		t.Errorf("Expected default TTL 48 hours, got %v", cfg.MessageTTL)
	}
}

func TestLoad_CustomPort(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Setenv("PORT", "8080")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.ServerPort != "8080" {
		t.Errorf("Expected port '8080', got '%s'", cfg.ServerPort)
	}
}

func TestLoad_CustomRedisDB(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Setenv("REDIS_DB", "5")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.RedisDB != 5 {
		t.Errorf("Expected Redis DB 5, got %d", cfg.RedisDB)
	}
}

func TestLoad_CustomMessageTTL(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Setenv("MESSAGE_TTL", "72")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.MessageTTL.Hours() != 72 {
		t.Errorf("Expected TTL 72 hours, got %v", cfg.MessageTTL)
	}
}

func TestLoad_ValidAdminID(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Setenv("ADMIN_ID", "123456789")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.AdminID != 123456789 {
		t.Errorf("Expected AdminID 123456789, got %d", cfg.AdminID)
	}
}

// ==================== Missing Required Fields Tests ====================

func TestLoad_MissingFactoryBotToken(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Unsetenv("FACTORY_BOT_TOKEN")

	_, err := config.Load()
	if err == nil {
		t.Error("Expected error for missing FACTORY_BOT_TOKEN")
	}

	if !strings.Contains(err.Error(), "FACTORY_BOT_TOKEN") {
		t.Errorf("Error should mention FACTORY_BOT_TOKEN, got: %v", err)
	}
}

func TestLoad_MissingWebhookURL(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Unsetenv("WEBHOOK_URL")

	_, err := config.Load()
	if err == nil {
		t.Error("Expected error for missing WEBHOOK_URL")
	}

	if !strings.Contains(err.Error(), "WEBHOOK_URL") {
		t.Errorf("Error should mention WEBHOOK_URL, got: %v", err)
	}
}

func TestLoad_MissingDBHost(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Unsetenv("DB_HOST")

	_, err := config.Load()
	if err == nil {
		t.Error("Expected error for missing DB_HOST")
	}
}

func TestLoad_MissingDBUser(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Unsetenv("DB_USER")

	_, err := config.Load()
	if err == nil {
		t.Error("Expected error for missing DB_USER")
	}
}

func TestLoad_MissingDBName(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Unsetenv("DB_NAME")

	_, err := config.Load()
	if err == nil {
		t.Error("Expected error for missing DB_NAME")
	}
}

func TestLoad_MissingRedisAddr(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Unsetenv("REDIS_ADDR")

	_, err := config.Load()
	if err == nil {
		t.Error("Expected error for missing REDIS_ADDR")
	}

	if !strings.Contains(err.Error(), "REDIS_ADDR") {
		t.Errorf("Error should mention REDIS_ADDR, got: %v", err)
	}
}

func TestLoad_MissingEncryptionKey(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Unsetenv("BOT_ENCRYPTION_KEY")

	_, err := config.Load()
	if err == nil {
		t.Error("Expected error for missing BOT_ENCRYPTION_KEY")
	}

	if !strings.Contains(err.Error(), "BOT_ENCRYPTION_KEY") {
		t.Errorf("Error should mention BOT_ENCRYPTION_KEY, got: %v", err)
	}
}

// ==================== Invalid Values Tests ====================

func TestLoad_InvalidAdminID(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Setenv("ADMIN_ID", "not-a-number")

	_, err := config.Load()
	if err == nil {
		t.Error("Expected error for invalid ADMIN_ID")
	}

	if !strings.Contains(err.Error(), "ADMIN_ID") {
		t.Errorf("Error should mention ADMIN_ID, got: %v", err)
	}
}

func TestLoad_InvalidRedisDB(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Setenv("REDIS_DB", "not-a-number")

	_, err := config.Load()
	if err == nil {
		t.Error("Expected error for invalid REDIS_DB")
	}

	if !strings.Contains(err.Error(), "REDIS_DB") {
		t.Errorf("Error should mention REDIS_DB, got: %v", err)
	}
}

func TestLoad_InvalidMessageTTL(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Setenv("MESSAGE_TTL", "not-a-number")

	_, err := config.Load()
	if err == nil {
		t.Error("Expected error for invalid MESSAGE_TTL")
	}

	if !strings.Contains(err.Error(), "MESSAGE_TTL") {
		t.Errorf("Error should mention MESSAGE_TTL, got: %v", err)
	}
}

func TestLoad_InvalidEncryptionKeyLength_Short(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Setenv("BOT_ENCRYPTION_KEY", "too-short")

	_, err := config.Load()
	if err == nil {
		t.Error("Expected error for encryption key < 32 bytes")
	}

	if !strings.Contains(err.Error(), "32 bytes") {
		t.Errorf("Error should mention 32 bytes, got: %v", err)
	}
}

func TestLoad_InvalidEncryptionKeyLength_Long(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Setenv("BOT_ENCRYPTION_KEY", strings.Repeat("x", 64))

	_, err := config.Load()
	if err == nil {
		t.Error("Expected error for encryption key > 32 bytes")
	}
}

// ==================== GetDSN Tests ====================

func TestGetDSN_Format(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	dsn := cfg.GetDSN()

	// Should contain user, password, host, and database
	if !strings.Contains(dsn, "root") {
		t.Error("DSN should contain username")
	}
	if !strings.Contains(dsn, "password") {
		t.Error("DSN should contain password")
	}
	if !strings.Contains(dsn, "localhost:3306") {
		t.Error("DSN should contain host")
	}
	if !strings.Contains(dsn, "testdb") {
		t.Error("DSN should contain database name")
	}
	if !strings.Contains(dsn, "parseTime=true") {
		t.Error("DSN should include parseTime=true")
	}
	if !strings.Contains(dsn, "utf8mb4") {
		t.Error("DSN should include utf8mb4 charset")
	}
}

func TestGetDSN_SpecialCharactersInPassword(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Setenv("DB_PASS", "p@ss:word/with?special=chars")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	dsn := cfg.GetDSN()

	// Password should be included as-is
	if !strings.Contains(dsn, "p@ss:word/with?special=chars") {
		t.Error("DSN should contain password with special characters")
	}
}

func TestGetDSN_EmptyPassword(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Setenv("DB_PASS", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	dsn := cfg.GetDSN()

	// Should still generate valid DSN format
	if !strings.Contains(dsn, "root:@tcp") {
		t.Error("DSN should handle empty password correctly")
	}
}

// ==================== Edge Cases ====================

func TestLoad_EmptyAdminID(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	// ADMIN_ID not set - should be 0 (optional field)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Expected no error for empty ADMIN_ID, got: %v", err)
	}

	if cfg.AdminID != 0 {
		t.Errorf("Expected AdminID 0 when not set, got %d", cfg.AdminID)
	}
}

func TestLoad_RedisPassword(t *testing.T) {
	clearEnv()
	defer clearEnv()
	setValidEnv()
	os.Setenv("REDIS_PASSWORD", "redis-secret")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.RedisPassword != "redis-secret" {
		t.Errorf("Expected Redis password 'redis-secret', got '%s'", cfg.RedisPassword)
	}
}
