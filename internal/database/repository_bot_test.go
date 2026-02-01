package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/Amr-9/botforge/internal/database"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

// ==================== Bot Management Tests ====================

func TestCreateBot_Extended(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")
	mysql := database.NewMySQLFromDB(sqlxDB)
	repo := database.NewRepository(mysql, "12345678901234567890123456789012")

	mock.ExpectExec("INSERT INTO bots").
		WithArgs(sqlmock.AnyArg(), int64(12345), "testbot").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	bot, err := repo.CreateBot(ctx, "123456789:ABCdef", int64(12345), "testbot")
	if err != nil {
		t.Fatalf("CreateBot failed: %v", err)
	}

	if bot == nil {
		t.Error("Expected bot to be returned")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestGetBotsByOwner_Extended(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")
	mysql := database.NewMySQLFromDB(sqlxDB)
	repo := database.NewRepository(mysql, "12345678901234567890123456789012")

	rows := sqlmock.NewRows([]string{"id", "token", "username", "owner_chat_id", "is_active", "start_message", "forward_auto_replies", "forced_sub_enabled", "forced_sub_message", "show_sent_confirmation", "created_at", "updated_at", "deleted_at"}).
		AddRow(1, "encrypted1", "bot1", 12345, true, "", true, false, "", true, time.Now(), time.Now(), nil).
		AddRow(2, "encrypted2", "bot2", 12345, false, "", true, false, "", true, time.Now(), time.Now(), nil)

	mock.ExpectQuery("SELECT (.+) FROM bots WHERE owner_chat_id").
		WithArgs(int64(12345)).
		WillReturnRows(rows)

	ctx := context.Background()
	bots, err := repo.GetBotsByOwner(ctx, int64(12345))
	if err != nil {
		t.Fatalf("GetBotsByOwner failed: %v", err)
	}

	if len(bots) != 2 {
		t.Errorf("Expected 2 bots, got %d", len(bots))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestDeleteBot_Extended(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")
	mysql := database.NewMySQLFromDB(sqlxDB)
	repo := database.NewRepository(mysql, "12345678901234567890123456789012")

	// DeleteBot uses token, not ID - and it's a soft delete (UPDATE)
	mock.ExpectExec("UPDATE bots SET deleted_at").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ctx := context.Background()
	err = repo.DeleteBot(ctx, "123456789:ABCdef")
	if err != nil {
		t.Fatalf("DeleteBot failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

// ==================== Ban Tests ====================

func TestBanUser_Extended(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")
	mysql := database.NewMySQLFromDB(sqlxDB)
	repo := database.NewRepository(mysql, "12345678901234567890123456789012")

	// BanUser takes botID, userChatID, bannedBy (not reason)
	mock.ExpectExec("INSERT INTO bans").
		WithArgs(int64(1), int64(99999), int64(12345)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	err = repo.BanUser(ctx, int64(1), int64(99999), int64(12345))
	if err != nil {
		t.Fatalf("BanUser failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestUnbanUser_Extended(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")
	mysql := database.NewMySQLFromDB(sqlxDB)
	repo := database.NewRepository(mysql, "12345678901234567890123456789012")

	mock.ExpectExec("DELETE FROM bans WHERE bot_id").
		WithArgs(int64(1), int64(99999)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ctx := context.Background()
	err = repo.UnbanUser(ctx, int64(1), int64(99999))
	if err != nil {
		t.Fatalf("UnbanUser failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestIsUserBanned_Extended(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")
	mysql := database.NewMySQLFromDB(sqlxDB)
	repo := database.NewRepository(mysql, "12345678901234567890123456789012")

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(int64(1), int64(99999)).
		WillReturnRows(rows)

	ctx := context.Background()
	banned, err := repo.IsUserBanned(ctx, int64(1), int64(99999))
	if err != nil {
		t.Fatalf("IsUserBanned failed: %v", err)
	}

	if !banned {
		t.Error("Expected user to be banned")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestGetBannedUserCount_Extended(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")
	mysql := database.NewMySQLFromDB(sqlxDB)
	repo := database.NewRepository(mysql, "12345678901234567890123456789012")

	rows := sqlmock.NewRows([]string{"count"}).AddRow(25)

	mock.ExpectQuery("SELECT COUNT(.+) FROM bans WHERE bot_id").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	ctx := context.Background()
	count, err := repo.GetBannedUserCount(ctx, int64(1))
	if err != nil {
		t.Fatalf("GetBannedUserCount failed: %v", err)
	}

	if count != 25 {
		t.Errorf("Expected 25, got %d", count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

// ==================== User Count Tests ====================

func TestGetUniqueUserCount_Extended(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")
	mysql := database.NewMySQLFromDB(sqlxDB)
	repo := database.NewRepository(mysql, "12345678901234567890123456789012")

	rows := sqlmock.NewRows([]string{"count"}).AddRow(150)

	mock.ExpectQuery("SELECT COUNT(.+) FROM").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	ctx := context.Background()
	count, err := repo.GetUniqueUserCount(ctx, int64(1))
	if err != nil {
		t.Fatalf("GetUniqueUserCount failed: %v", err)
	}

	if count != 150 {
		t.Errorf("Expected 150, got %d", count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

// ==================== Message Stats Tests ====================

func TestGetTotalMessageCount_Extended(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")
	mysql := database.NewMySQLFromDB(sqlxDB)
	repo := database.NewRepository(mysql, "12345678901234567890123456789012")

	rows := sqlmock.NewRows([]string{"count"}).AddRow(1500)

	mock.ExpectQuery("SELECT COUNT(.+) FROM messages").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	ctx := context.Background()
	count, err := repo.GetTotalMessageCount(ctx, int64(1))
	if err != nil {
		t.Fatalf("GetTotalMessageCount failed: %v", err)
	}

	if count != 1500 {
		t.Errorf("Expected 1500, got %d", count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestGetMessageCountSince_Extended(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")
	mysql := database.NewMySQLFromDB(sqlxDB)
	repo := database.NewRepository(mysql, "12345678901234567890123456789012")

	rows := sqlmock.NewRows([]string{"count"}).AddRow(42)

	mock.ExpectQuery("SELECT COUNT(.+) FROM messages").
		WithArgs(int64(1), sqlmock.AnyArg()).
		WillReturnRows(rows)

	ctx := context.Background()
	since := time.Now().Add(-24 * time.Hour)
	count, err := repo.GetMessageCountSince(ctx, int64(1), since)
	if err != nil {
		t.Fatalf("GetMessageCountSince failed: %v", err)
	}

	if count != 42 {
		t.Errorf("Expected 42, got %d", count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

// ==================== Activate/Deactivate Tests ====================

func TestDeactivateBot(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")
	mysql := database.NewMySQLFromDB(sqlxDB)
	repo := database.NewRepository(mysql, "12345678901234567890123456789012")

	mock.ExpectExec("UPDATE bots SET is_active").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ctx := context.Background()
	err = repo.DeactivateBot(ctx, "123456789:ABCdef")
	if err != nil {
		t.Fatalf("DeactivateBot failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestActivateBot(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")
	mysql := database.NewMySQLFromDB(sqlxDB)
	repo := database.NewRepository(mysql, "12345678901234567890123456789012")

	mock.ExpectExec("UPDATE bots SET is_active").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ctx := context.Background()
	err = repo.ActivateBot(ctx, "123456789:ABCdef")
	if err != nil {
		t.Fatalf("ActivateBot failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}
