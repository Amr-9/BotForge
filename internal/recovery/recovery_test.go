package recovery_test

import (
	"sync"
	"testing"
	"time"

	"github.com/Amr-9/botforge/internal/recovery"
)

// TestRestartPolicy_ShouldRestart verifies basic restart policy behavior
func TestRestartPolicy_ShouldRestart(t *testing.T) {
	policy := recovery.NewRestartPolicy(3, 100*time.Millisecond, 1*time.Second)

	// First 3 attempts should succeed
	for i := 0; i < 3; i++ {
		shouldRestart, delay := policy.ShouldRestart()
		if !shouldRestart {
			t.Errorf("Attempt %d: expected shouldRestart=true, got false", i+1)
		}
		if delay <= 0 {
			t.Errorf("Attempt %d: expected positive delay, got %v", i+1, delay)
		}
	}

	// 4th attempt should fail (max retries exhausted)
	shouldRestart, _ := policy.ShouldRestart()
	if shouldRestart {
		t.Error("Expected shouldRestart=false after max retries exhausted")
	}
}

// TestRestartPolicy_ExponentialBackoff verifies exponential backoff calculation
func TestRestartPolicy_ExponentialBackoff(t *testing.T) {
	baseDelay := 100 * time.Millisecond
	maxDelay := 1 * time.Second
	policy := recovery.NewRestartPolicy(5, baseDelay, maxDelay)

	expectedDelays := []time.Duration{
		100 * time.Millisecond, // 100ms * 2^0
		200 * time.Millisecond, // 100ms * 2^1
		400 * time.Millisecond, // 100ms * 2^2
		800 * time.Millisecond, // 100ms * 2^3
		1 * time.Second,        // capped at maxDelay
	}

	for i, expected := range expectedDelays {
		_, delay := policy.ShouldRestart()
		if delay != expected {
			t.Errorf("Attempt %d: expected delay %v, got %v", i+1, expected, delay)
		}
	}
}

// TestRestartPolicy_Reset verifies retry counter reset
func TestRestartPolicy_Reset(t *testing.T) {
	policy := recovery.NewRestartPolicy(2, 100*time.Millisecond, 1*time.Second)

	// Use up retries
	policy.ShouldRestart()
	policy.ShouldRestart()

	if policy.GetRetryCount() != 2 {
		t.Errorf("Expected retry count to be 2, got %d", policy.GetRetryCount())
	}

	// Reset
	policy.Reset()

	if policy.GetRetryCount() != 0 {
		t.Errorf("Expected retry count to be 0 after reset, got %d", policy.GetRetryCount())
	}

	// Should be able to restart again
	shouldRestart, _ := policy.ShouldRestart()
	if !shouldRestart {
		t.Error("Expected shouldRestart=true after reset")
	}
}

// TestRestartPolicy_ThreadSafety verifies thread-safe access
func TestRestartPolicy_ThreadSafety(t *testing.T) {
	policy := recovery.NewRestartPolicy(100, 1*time.Millisecond, 10*time.Millisecond)

	var wg sync.WaitGroup
	goroutines := 10
	callsPerGoroutine := 10

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				policy.ShouldRestart()
			}
		}()
	}

	wg.Wait()

	expectedCount := goroutines * callsPerGoroutine
	if policy.GetRetryCount() != expectedCount {
		t.Errorf("Expected retry count %d, got %d", expectedCount, policy.GetRetryCount())
	}
}

// TestSafeGo_RecoversPanic verifies panic recovery in SafeGo
func TestSafeGo_RecoversPanic(t *testing.T) {
	recovered := make(chan bool, 1)

	recovery.SafeGo(
		func() { panic("test panic") },
		map[string]string{"test": "value"},
		func(info recovery.PanicInfo) {
			if info.Value != "test panic" {
				t.Errorf("Expected panic value 'test panic', got %v", info.Value)
			}
			if info.Context["test"] != "value" {
				t.Errorf("Expected context test=value, got %v", info.Context)
			}
			if info.StackTrace == "" {
				t.Error("Expected non-empty stack trace")
			}
			recovered <- true
		},
	)

	select {
	case <-recovered:
		// Test passed
	case <-time.After(2 * time.Second):
		t.Error("Panic was not recovered within timeout")
	}
}

// TestSafeGo_NoPanic verifies SafeGo works normally without panic
func TestSafeGo_NoPanic(t *testing.T) {
	completed := make(chan bool, 1)

	recovery.SafeGo(
		func() { completed <- true },
		map[string]string{"test": "nopanic"},
		nil, // Use default handler
	)

	select {
	case <-completed:
		// Test passed
	case <-time.After(2 * time.Second):
		t.Error("Function did not complete within timeout")
	}
}

// TestRestartController_Stop verifies controller stops the loop
func TestRestartController_Stop(t *testing.T) {
	controller := recovery.NewRestartController()

	// Check context is not done initially
	select {
	case <-controller.Context().Done():
		t.Error("Context should not be done initially")
	default:
		// Expected
	}

	// Stop the controller
	controller.Stop()

	// Check context is done after stop
	select {
	case <-controller.Context().Done():
		// Expected
	default:
		t.Error("Context should be done after Stop()")
	}
}

// TestPanicInfo_Fields verifies PanicInfo captures all fields correctly
func TestPanicInfo_Fields(t *testing.T) {
	now := time.Now()
	info := recovery.PanicInfo{
		Timestamp:  now,
		Value:      "test error",
		StackTrace: "stack trace here",
		Context:    map[string]string{"key": "value"},
	}

	if info.Timestamp != now {
		t.Errorf("Timestamp mismatch")
	}
	if info.Value != "test error" {
		t.Errorf("Value mismatch")
	}
	if info.StackTrace != "stack trace here" {
		t.Errorf("StackTrace mismatch")
	}
	if info.Context["key"] != "value" {
		t.Errorf("Context mismatch")
	}
}

// TestDefaultHandler_NoError verifies DefaultHandler doesn't panic
func TestDefaultHandler_NoError(t *testing.T) {
	// Should not panic
	info := recovery.PanicInfo{
		Timestamp:  time.Now(),
		Value:      "test",
		StackTrace: "stack",
		Context:    map[string]string{},
	}
	recovery.DefaultHandler(info) // Should complete without error
}
