package models_test

import (
	"testing"
	"time"

	"github.com/Amr-9/botforge/internal/models"
)

// ==================== Bot Model Tests ====================

func TestBot_Fields(t *testing.T) {
	now := time.Now()
	bot := models.Bot{
		ID:                   1,
		Token:                "123456:ABC",
		Username:             "testbot",
		OwnerChatID:          12345,
		IsActive:             true,
		StartMessage:         "Welcome!",
		ForwardAutoReplies:   true,
		ForcedSubEnabled:     false,
		ForcedSubMessage:     "",
		ShowSentConfirmation: true,
		CreatedAt:            now,
	}

	if bot.ID != 1 {
		t.Error("ID mismatch")
	}
	if bot.Token != "123456:ABC" {
		t.Error("Token mismatch")
	}
	if bot.Username != "testbot" {
		t.Error("Username mismatch")
	}
	if bot.OwnerChatID != 12345 {
		t.Error("OwnerChatID mismatch")
	}
	if !bot.IsActive {
		t.Error("IsActive should be true")
	}
	if bot.StartMessage != "Welcome!" {
		t.Error("StartMessage mismatch")
	}
	if !bot.ForwardAutoReplies {
		t.Error("ForwardAutoReplies should be true")
	}
	if bot.ForcedSubEnabled {
		t.Error("ForcedSubEnabled should be false")
	}
	if !bot.ShowSentConfirmation {
		t.Error("ShowSentConfirmation should be true")
	}
}

// ==================== MessageLog Model Tests ====================

func TestMessageLog_Fields(t *testing.T) {
	now := time.Now()
	log := models.MessageLog{
		ID:         1,
		AdminMsgID: 100,
		UserChatID: 12345678,
		BotID:      1,
		CreatedAt:  now,
	}

	if log.ID != 1 {
		t.Error("ID mismatch")
	}
	if log.AdminMsgID != 100 {
		t.Error("AdminMsgID mismatch")
	}
	if log.UserChatID != 12345678 {
		t.Error("UserChatID mismatch")
	}
	if log.BotID != 1 {
		t.Error("BotID mismatch")
	}
}

// ==================== BannedUser Model Tests ====================

func TestBannedUser_Fields(t *testing.T) {
	now := time.Now()
	banned := models.BannedUser{
		ID:         1,
		BotID:      1,
		UserChatID: 99999,
		BannedBy:   12345,
		CreatedAt:  now,
	}

	if banned.BannedBy != 12345 {
		t.Error("BannedBy mismatch")
	}
	if banned.UserChatID != 99999 {
		t.Error("UserChatID mismatch")
	}
}

// ==================== AutoReply Model Tests ====================

func TestAutoReply_TextReply(t *testing.T) {
	reply := models.AutoReply{
		ID:          1,
		BotID:       1,
		TriggerWord: "hello",
		Response:    "Hi there!",
		MessageType: "text",
		TriggerType: "keyword",
		MatchType:   "contains",
		IsActive:    true,
	}

	if reply.TriggerWord != "hello" {
		t.Error("TriggerWord mismatch")
	}
	if reply.Response != "Hi there!" {
		t.Error("Response mismatch")
	}
	if reply.MessageType != "text" {
		t.Error("MessageType should be 'text'")
	}
}

func TestAutoReply_MediaReply(t *testing.T) {
	reply := models.AutoReply{
		ID:          2,
		BotID:       1,
		TriggerWord: "photo",
		Response:    "",
		MessageType: "photo",
		FileID:      "AgACAgIAAxkBAAI...",
		Caption:     "Beautiful image!",
		TriggerType: "command",
		MatchType:   "exact",
		IsActive:    true,
	}

	if reply.FileID == "" {
		t.Error("FileID should not be empty for media")
	}
	if reply.MessageType != "photo" {
		t.Error("MessageType should be 'photo'")
	}
	if reply.TriggerType != "command" {
		t.Error("TriggerType should be 'command'")
	}
}

// ==================== ForcedChannel Model Tests ====================

func TestForcedChannel_Fields(t *testing.T) {
	channel := models.ForcedChannel{
		ID:              1,
		BotID:           1,
		ChannelID:       -1001234567890,
		ChannelUsername: "testchannel",
		ChannelTitle:    "Test Channel",
		InviteLink:      "https://t.me/+abc123",
		IsActive:        true,
	}

	if channel.ChannelID != -1001234567890 {
		t.Error("ChannelID mismatch")
	}
	if channel.ChannelUsername != "testchannel" {
		t.Error("ChannelUsername mismatch")
	}
	if channel.InviteLink != "https://t.me/+abc123" {
		t.Error("InviteLink mismatch")
	}
}

// ==================== ScheduledMessage Model Tests ====================

func TestScheduledMessage_TextMessage(t *testing.T) {
	now := time.Now()
	msg := models.ScheduledMessage{
		ID:            1,
		BotID:         1,
		OwnerChatID:   12345,
		MessageType:   models.MessageTypeText,
		MessageText:   "Daily reminder!",
		ScheduleType:  models.ScheduleTypeDaily,
		ScheduledTime: now,
		TimeOfDay:     "09:00",
		Status:        models.ScheduleStatusPending,
		CreatedAt:     now,
	}

	if msg.MessageType != "text" {
		t.Error("MessageType should be 'text'")
	}
	if msg.ScheduleType != "daily" {
		t.Error("ScheduleType should be 'daily'")
	}
	if msg.Status != "pending" {
		t.Error("Status should be 'pending'")
	}
}

func TestScheduledMessage_WeeklyWithDayOfWeek(t *testing.T) {
	dayOfWeek := 1 // Monday
	msg := models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeWeekly,
		TimeOfDay:    "10:00",
		DayOfWeek:    &dayOfWeek,
	}

	if msg.DayOfWeek == nil {
		t.Error("DayOfWeek should not be nil for weekly schedule")
	}
	if *msg.DayOfWeek != 1 {
		t.Errorf("DayOfWeek should be 1, got %d", *msg.DayOfWeek)
	}
}

func TestScheduledMessage_MediaMessage(t *testing.T) {
	msg := models.ScheduledMessage{
		MessageType: models.MessageTypePhoto,
		FileID:      "Photo123",
		Caption:     "Check this out!",
	}

	if msg.FileID == "" {
		t.Error("FileID should not be empty")
	}
	if msg.Caption == "" {
		t.Error("Caption should not be empty")
	}
}

func TestScheduledMessage_StatusTransitions(t *testing.T) {
	statuses := []string{
		models.ScheduleStatusPending,
		models.ScheduleStatusSent,
		models.ScheduleStatusFailed,
		models.ScheduleStatusPaused,
		models.ScheduleStatusCancelled,
	}

	expected := []string{"pending", "sent", "failed", "paused", "cancelled"}

	for i, status := range statuses {
		if status != expected[i] {
			t.Errorf("Status %d mismatch: expected '%s', got '%s'", i, expected[i], status)
		}
	}
}

// ==================== Model Type Constants Tests ====================

func TestScheduleTypeConstants(t *testing.T) {
	tests := map[string]string{
		models.ScheduleTypeOnce:   "once",
		models.ScheduleTypeDaily:  "daily",
		models.ScheduleTypeWeekly: "weekly",
	}

	for got, expected := range tests {
		if got != expected {
			t.Errorf("Expected '%s', got '%s'", expected, got)
		}
	}
}

func TestMessageTypeConstants(t *testing.T) {
	tests := map[string]string{
		models.MessageTypeText:     "text",
		models.MessageTypePhoto:    "photo",
		models.MessageTypeVideo:    "video",
		models.MessageTypeDocument: "document",
	}

	for got, expected := range tests {
		if got != expected {
			t.Errorf("Expected '%s', got '%s'", expected, got)
		}
	}
}

func TestScheduleStatusConstants(t *testing.T) {
	tests := map[string]string{
		models.ScheduleStatusPending:   "pending",
		models.ScheduleStatusSent:      "sent",
		models.ScheduleStatusFailed:    "failed",
		models.ScheduleStatusPaused:    "paused",
		models.ScheduleStatusCancelled: "cancelled",
	}

	for got, expected := range tests {
		if got != expected {
			t.Errorf("Expected '%s', got '%s'", expected, got)
		}
	}
}
