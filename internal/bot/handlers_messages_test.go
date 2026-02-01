package bot

import (
	"testing"
	"time"

	"gopkg.in/telebot.v3"
)

// ==================== formatUserInfo Tests ====================

func TestFormatUserInfo_FullUser(t *testing.T) {
	user := &telebot.User{
		ID:        123456789,
		FirstName: "John",
		LastName:  "Doe",
		Username:  "johndoe",
	}

	result := formatUserInfo(user)

	// Check it contains the expected elements
	if result == "" {
		t.Error("Expected non-empty result")
	}

	// Should contain the user's first name
	if !contains(result, "John") {
		t.Error("Expected result to contain first name 'John'")
	}

	// Should contain the user's last name
	if !contains(result, "Doe") {
		t.Error("Expected result to contain last name 'Doe'")
	}

	// Should contain the username
	if !contains(result, "@johndoe") {
		t.Error("Expected result to contain username '@johndoe'")
	}

	// Should contain the ID
	if !contains(result, "123456789") {
		t.Error("Expected result to contain user ID")
	}
}

func TestFormatUserInfo_NoLastName(t *testing.T) {
	user := &telebot.User{
		ID:        987654321,
		FirstName: "Alice",
		Username:  "alice123",
	}

	result := formatUserInfo(user)

	if !contains(result, "Alice") {
		t.Error("Expected result to contain first name")
	}
	if !contains(result, "@alice123") {
		t.Error("Expected result to contain username")
	}
}

func TestFormatUserInfo_NoUsername(t *testing.T) {
	user := &telebot.User{
		ID:        111222333,
		FirstName: "Bob",
		LastName:  "Smith",
	}

	result := formatUserInfo(user)

	if !contains(result, "Bob") {
		t.Error("Expected result to contain first name")
	}
	if !contains(result, "Smith") {
		t.Error("Expected result to contain last name")
	}
	// Should not contain @ since no username
	if contains(result, "@\n") {
		t.Error("Should not have orphaned @ symbol")
	}
}

func TestFormatUserInfo_OnlyFirstName(t *testing.T) {
	user := &telebot.User{
		ID:        444555666,
		FirstName: "Charlie",
	}

	result := formatUserInfo(user)

	if !contains(result, "Charlie") {
		t.Error("Expected result to contain first name")
	}
	if !contains(result, "444555666") {
		t.Error("Expected result to contain user ID")
	}
}

// ==================== formatInt64 Tests ====================

func TestFormatInt64_Positive(t *testing.T) {
	result := formatInt64(123456789)
	if result != "123456789" {
		t.Errorf("Expected '123456789', got '%s'", result)
	}
}

func TestFormatInt64_Zero(t *testing.T) {
	result := formatInt64(0)
	if result != "0" {
		t.Errorf("Expected '0', got '%s'", result)
	}
}

func TestFormatInt64_Negative(t *testing.T) {
	result := formatInt64(-987654321)
	if result != "-987654321" {
		t.Errorf("Expected '-987654321', got '%s'", result)
	}
}

func TestFormatInt64_LargeNumber(t *testing.T) {
	result := formatInt64(9007199254740991) // Max safe integer in JS
	if result != "9007199254740991" {
		t.Errorf("Expected '9007199254740991', got '%s'", result)
	}
}

// ==================== todayStart Tests ====================

func TestTodayStart_CurrentDay(t *testing.T) {
	// Save original timeNow
	originalTimeNow := timeNow
	defer func() { timeNow = originalTimeNow }()

	// Mock timeNow to return a specific time
	mockTime := time.Date(2026, 2, 1, 14, 30, 45, 123456789, time.UTC)
	timeNow = func() time.Time { return mockTime }

	result := todayStart()

	expected := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestTodayStart_Midnight(t *testing.T) {
	originalTimeNow := timeNow
	defer func() { timeNow = originalTimeNow }()

	// Mock at exactly midnight
	mockTime := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return mockTime }

	result := todayStart()

	if !result.Equal(mockTime) {
		t.Errorf("Expected %v, got %v", mockTime, result)
	}
}

func TestTodayStart_EndOfDay(t *testing.T) {
	originalTimeNow := timeNow
	defer func() { timeNow = originalTimeNow }()

	// Mock at 23:59:59
	mockTime := time.Date(2026, 2, 1, 23, 59, 59, 999999999, time.UTC)
	timeNow = func() time.Time { return mockTime }

	result := todayStart()

	expected := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestTodayStart_DifferentTimezone(t *testing.T) {
	originalTimeNow := timeNow
	defer func() { timeNow = originalTimeNow }()

	// Use a different timezone
	loc, _ := time.LoadLocation("America/New_York")
	mockTime := time.Date(2026, 2, 1, 15, 30, 0, 0, loc)
	timeNow = func() time.Time { return mockTime }

	result := todayStart()

	// Should preserve the timezone
	if result.Location().String() != loc.String() {
		t.Errorf("Expected timezone %s, got %s", loc.String(), result.Location().String())
	}

	// Should be midnight on the same day
	if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 {
		t.Error("Expected midnight (00:00:00)")
	}
}

// ==================== Helper Functions ====================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
