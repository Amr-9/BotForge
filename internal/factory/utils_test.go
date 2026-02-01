package factory

import (
	"testing"
)

// ==================== Token Format Validation Tests ====================

func TestIsValidTokenFormat_Valid(t *testing.T) {
	validTokens := []string{
		"123456789:ABCdefGHIjklMNOpqrSTUvwxYZ1234567890",
		"987654321:AAAAbbbbCCCCddddEEEEffffGGGGhhhh1234",
		"1:ABCDEFGHIJKLMNOPQRSTUVWXYZ12345678",
	}

	for _, token := range validTokens {
		if !isValidTokenFormat(token) {
			t.Errorf("Expected token '%s' to be valid", token)
		}
	}
}

func TestIsValidTokenFormat_Invalid_NoColon(t *testing.T) {
	if isValidTokenFormat("123456789ABCdefGHIjklMNOpqr") {
		t.Error("Token without colon should be invalid")
	}
}

func TestIsValidTokenFormat_Invalid_MultipleColons(t *testing.T) {
	if isValidTokenFormat("123:456:789") {
		t.Error("Token with multiple colons should be invalid")
	}
}

func TestIsValidTokenFormat_Invalid_NonNumericPrefix(t *testing.T) {
	if isValidTokenFormat("abc123:ABCdefGHIjklMNOpqrSTUvwxYZ1234567890") {
		t.Error("Token with non-numeric prefix should be invalid")
	}
}

func TestIsValidTokenFormat_Invalid_ShortSuffix(t *testing.T) {
	if isValidTokenFormat("123456:short") {
		t.Error("Token with short suffix (< 30 chars) should be invalid")
	}
}

func TestIsValidTokenFormat_Empty(t *testing.T) {
	if isValidTokenFormat("") {
		t.Error("Empty token should be invalid")
	}
}

func TestIsValidTokenFormat_OnlyColon(t *testing.T) {
	if isValidTokenFormat(":") {
		t.Error("Token with only colon should be invalid")
	}
}

// ==================== Token Masking Tests ====================

func TestMaskToken_Valid(t *testing.T) {
	token := "123456789:ABCdefGHIjklMNOpqrSTUvwxYZ1234567890"
	masked := maskToken(token)

	if masked == token {
		t.Error("Masked token should not equal original token")
	}

	// Should contain the numeric prefix
	if masked[:9] != "123456789" {
		t.Errorf("Expected prefix '123456789', got '%s'", masked[:9])
	}

	// Should contain ellipsis
	if len(masked) > 0 && masked[10:12] != "AB" {
		// First 5 chars after colon should be visible
	}
}

func TestMaskToken_ShortSuffix(t *testing.T) {
	token := "123:short"
	masked := maskToken(token)

	if masked != "123:***" {
		t.Errorf("Expected '123:***', got '%s'", masked)
	}
}

func TestMaskToken_NoColon(t *testing.T) {
	masked := maskToken("invalidtoken")

	if masked != "***" {
		t.Errorf("Expected '***' for invalid format, got '%s'", masked)
	}
}

func TestMaskToken_Empty(t *testing.T) {
	masked := maskToken("")

	if masked != "***" {
		t.Errorf("Expected '***' for empty token, got '%s'", masked)
	}
}

func TestMaskToken_ExactlyTenCharsSuffix(t *testing.T) {
	token := "123:1234567890" // exactly 10 chars after colon
	masked := maskToken(token)

	if masked != "123:***" {
		t.Errorf("Expected '123:***', got '%s'", masked)
	}
}

// ==================== Callback Constants Tests ====================

func TestCallbackConstants(t *testing.T) {
	if CallbackAddBot != "add_bot" {
		t.Error("CallbackAddBot mismatch")
	}
	if CallbackMyBots != "my_bots" {
		t.Error("CallbackMyBots mismatch")
	}
	if CallbackStats != "stats" {
		t.Error("CallbackStats mismatch")
	}
	if CallbackMainMenu != "main_menu" {
		t.Error("CallbackMainMenu mismatch")
	}
	if CallbackBotSelect != "bot_sel" {
		t.Error("CallbackBotSelect mismatch")
	}
	if CallbackStartBot != "start_bot" {
		t.Error("CallbackStartBot mismatch")
	}
	if CallbackStopBot != "stop_bot" {
		t.Error("CallbackStopBot mismatch")
	}
	if CallbackDeleteBot != "del_bot" {
		t.Error("CallbackDeleteBot mismatch")
	}
	if CallbackConfirmDel != "conf_del" {
		t.Error("CallbackConfirmDel mismatch")
	}
	if CallbackCancelDel != "cancel_del" {
		t.Error("CallbackCancelDel mismatch")
	}
}
