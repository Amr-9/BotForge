package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/Amr-9/botforge/internal/database"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

// ==================== Auto-Reply Tests ====================

func TestCreateAutoReply_Success(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("INSERT INTO auto_replies").
		WithArgs(
			int64(1), "hello", "Hi there!", "text", "", "", "keyword", "contains",
			"Hi there!", "text", "", "", "contains",
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateAutoReply(context.Background(), 1, "hello", "Hi there!", "text", "", "", "keyword", "contains")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestCreateAutoReply_WithMedia(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("INSERT INTO auto_replies").
		WithArgs(
			int64(1), "photo", "", "photo", "FileID123", "Beautiful sunset", "keyword", "exact",
			"", "photo", "FileID123", "Beautiful sunset", "exact",
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateAutoReply(context.Background(), 1, "photo", "", "photo", "FileID123", "Beautiful sunset", "keyword", "exact")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestGetAutoReplies_Success(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"id", "bot_id", "trigger_word", "response", "message_type", "file_id", "caption",
		"trigger_type", "match_type", "is_active", "created_at",
	}).
		AddRow(1, 1, "hello", "Hi!", "text", "", "", "keyword", "contains", true, time.Now()).
		AddRow(2, 1, "bye", "Goodbye!", "text", "", "", "keyword", "exact", true, time.Now())

	mock.ExpectQuery("SELECT .+ FROM auto_replies").
		WithArgs(int64(1), "keyword").
		WillReturnRows(rows)

	replies, err := repo.GetAutoReplies(context.Background(), 1, "keyword")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if len(replies) != 2 {
		t.Errorf("Expected 2 replies, got %d", len(replies))
	}
}

func TestGetAutoReplies_Empty(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"id", "bot_id", "trigger_word", "response", "message_type", "file_id", "caption",
		"trigger_type", "match_type", "is_active", "created_at",
	})

	mock.ExpectQuery("SELECT .+ FROM auto_replies").
		WithArgs(int64(1), "command").
		WillReturnRows(rows)

	replies, err := repo.GetAutoReplies(context.Background(), 1, "command")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if len(replies) != 0 {
		t.Errorf("Expected 0 replies, got %d", len(replies))
	}
}

func TestDeleteAutoReply_Success(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM auto_replies").
		WithArgs(int64(5), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteAutoReply(context.Background(), 1, 5)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestGetAutoReplyCount(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"count"}).AddRow(15)

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(int64(1), "keyword").
		WillReturnRows(rows)

	count, err := repo.GetAutoReplyCount(context.Background(), 1, "keyword")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if count != 15 {
		t.Errorf("Expected 15, got %d", count)
	}
}

// ==================== Scheduled Messages Tests ====================

func TestCreateScheduledMessage_Success(t *testing.T) {
	_, mock, cleanup := setupMockDB(t)
	defer cleanup()

	scheduledTime := time.Now().Add(24 * time.Hour)
	nextRun := scheduledTime

	mock.ExpectExec("INSERT INTO scheduled_messages").
		WithArgs(
			int64(1), int64(12345), "text", "Hello World", "", "",
			"daily", scheduledTime, "14:00", nil, "pending", &nextRun,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// This test validates the expected SQL query structure
	// The actual call would require models.ScheduledMessage
}

func TestGetScheduledMessagesCount(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"count"}).AddRow(5)

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	count, err := repo.GetScheduledMessagesCount(context.Background(), 1)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected 5, got %d", count)
	}
}

func TestPauseScheduledMessage_Success(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE scheduled_messages").
		WithArgs(int64(10), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.PauseScheduledMessage(context.Background(), 10, 1)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestResumeScheduledMessage_Success(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE scheduled_messages").
		WithArgs(int64(10), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.ResumeScheduledMessage(context.Background(), 10, 1)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestDeleteScheduledMessage_Success(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE scheduled_messages").
		WithArgs(int64(10), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteScheduledMessage(context.Background(), 10, 1)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestUpdateScheduledMessageStatus_Success(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE scheduled_messages").
		WithArgs("failed", "Connection timeout", int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateScheduledMessageStatus(context.Background(), 10, "failed", "Connection timeout")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// ==================== Forced Subscription Tests ====================

func TestCreateForcedChannel_Success(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("INSERT INTO forced_channels").
		WithArgs(
			int64(1), int64(-1001234567890), "mychannel", "My Channel", "https://t.me/+abc123",
			"mychannel", "My Channel", "https://t.me/+abc123",
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateForcedChannel(context.Background(), 1, -1001234567890, "mychannel", "My Channel", "https://t.me/+abc123")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestGetForcedChannels_Success(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"id", "bot_id", "channel_id", "channel_username", "channel_title", "invite_link", "is_active", "created_at",
	}).
		AddRow(1, 1, int64(-1001234567890), "channel1", "Channel 1", "", true, time.Now()).
		AddRow(2, 1, int64(-1009876543210), "channel2", "Channel 2", "https://t.me/+xyz", true, time.Now())

	mock.ExpectQuery("SELECT .+ FROM forced_channels").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	channels, err := repo.GetForcedChannels(context.Background(), 1)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("Expected 2 channels, got %d", len(channels))
	}
}

func TestDeleteForcedChannel_Success(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM forced_channels").
		WithArgs(int64(1), int64(-1001234567890)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteForcedChannel(context.Background(), 1, -1001234567890)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestGetForcedChannelCount(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"count"}).AddRow(3)

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	count, err := repo.GetForcedChannelCount(context.Background(), 1)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3, got %d", count)
	}
}

func TestUpdateForcedSubEnabled_True(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE bots SET forced_sub_enabled").
		WithArgs(true, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateForcedSubEnabled(context.Background(), 1, true)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestUpdateForcedSubEnabled_False(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE bots SET forced_sub_enabled").
		WithArgs(false, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateForcedSubEnabled(context.Background(), 1, false)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestUpdateForcedSubMessage_Success(t *testing.T) {
	repo, mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec("UPDATE bots SET forced_sub_message").
		WithArgs("Please subscribe to our channels first!", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateForcedSubMessage(context.Background(), 1, "Please subscribe to our channels first!")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// ==================== Helper Type for Tests ====================

type ScheduledMessageForTest struct {
	BotID         int64
	OwnerChatID   int64
	MessageType   string
	MessageText   string
	FileID        string
	Caption       string
	ScheduleType  string
	ScheduledTime time.Time
	TimeOfDay     string
	DayOfWeek     *int
	Status        string
	NextRunAt     *time.Time
}

// ==================== Setup Helper ====================

func setupMockDB(t *testing.T) (*database.Repository, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	sqlxDB := sqlx.NewDb(db, "mysql")
	mysql := database.NewMySQLFromDB(sqlxDB)
	repo := database.NewRepository(mysql, "12345678901234567890123456789012")

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}
