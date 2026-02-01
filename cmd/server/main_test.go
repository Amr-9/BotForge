package main

import (
	"testing"
)

// ==================== maskToken Tests ====================

func TestMaskToken_Valid(t *testing.T) {
	token := "123456789:ABCdefGHIjklMNOpqr"
	masked := maskToken(token)

	expected := "123456789:..."
	if masked != expected {
		t.Errorf("Expected '%s', got '%s'", expected, masked)
	}
}

func TestMaskToken_Short(t *testing.T) {
	token := "short"
	masked := maskToken(token)

	if masked != "***" {
		t.Errorf("Expected '***' for short token, got '%s'", masked)
	}
}

func TestMaskToken_ExactlyFifteen(t *testing.T) {
	token := "123456789012345" // exactly 15 chars
	masked := maskToken(token)

	// Tokens need more than 15 chars (>= 16) to show partial masking
	// Exactly 15 chars returns fully masked
	if masked != "***" {
		t.Errorf("Expected '***' for exactly 15 chars, got '%s'", masked)
	}
}

func TestMaskToken_Empty(t *testing.T) {
	masked := maskToken("")

	if masked != "***" {
		t.Errorf("Expected '***' for empty token, got '%s'", masked)
	}
}

func TestMaskToken_JustAboveFifteen(t *testing.T) {
	token := "1234567890123456" // 16 chars
	masked := maskToken(token)

	if masked != "1234567890..." {
		t.Errorf("Expected '1234567890...', got '%s'", masked)
	}
}
