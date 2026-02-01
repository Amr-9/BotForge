package bot

import (
	"testing"

	"gopkg.in/telebot.v3"
)

// ==================== formatBanUserName Tests ====================

func TestFormatBanUserName_FullInfo(t *testing.T) {
	chat := &telebot.Chat{
		FirstName: "John",
		LastName:  "Doe",
		Username:  "johndoe",
	}

	result := formatBanUserName(chat)

	if result != "John Doe (@johndoe)" {
		t.Errorf("Expected 'John Doe (@johndoe)', got '%s'", result)
	}
}

func TestFormatBanUserName_NoLastName(t *testing.T) {
	chat := &telebot.Chat{
		FirstName: "Alice",
		Username:  "alice123",
	}

	result := formatBanUserName(chat)

	if result != "Alice (@alice123)" {
		t.Errorf("Expected 'Alice (@alice123)', got '%s'", result)
	}
}

func TestFormatBanUserName_NoUsername(t *testing.T) {
	chat := &telebot.Chat{
		FirstName: "Bob",
		LastName:  "Smith",
	}

	result := formatBanUserName(chat)

	if result != "Bob Smith" {
		t.Errorf("Expected 'Bob Smith', got '%s'", result)
	}
}

func TestFormatBanUserName_OnlyFirstName(t *testing.T) {
	chat := &telebot.Chat{
		FirstName: "Charlie",
	}

	result := formatBanUserName(chat)

	if result != "Charlie" {
		t.Errorf("Expected 'Charlie', got '%s'", result)
	}
}

func TestFormatBanUserName_EmptyChat(t *testing.T) {
	chat := &telebot.Chat{}

	result := formatBanUserName(chat)

	// Should return empty string for empty chat
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestFormatBanUserName_OnlyUsername(t *testing.T) {
	chat := &telebot.Chat{
		Username: "someuser",
	}

	result := formatBanUserName(chat)

	// FirstName is empty, so should just have username part
	if result != " (@someuser)" {
		t.Errorf("Expected ' (@someuser)', got '%s'", result)
	}
}
