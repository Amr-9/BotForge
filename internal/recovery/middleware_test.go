package recovery_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Amr-9/botforge/internal/recovery"
)

// ==================== HTTPMiddleware Tests ====================

func TestHTTPMiddleware_NoPanic(t *testing.T) {
	// Create a simple handler that doesn't panic
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with recovery middleware
	wrapped := recovery.HTTPMiddleware(handler, nil)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Execute request
	wrapped.ServeHTTP(rr, req)

	// Verify response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	if rr.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", rr.Body.String())
	}
}

func TestHTTPMiddleware_WithPanic(t *testing.T) {
	recovered := make(chan bool, 1)

	// Create a handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic in handler")
	})

	// Custom recovery handler
	customHandler := func(info recovery.PanicInfo) {
		if info.Value != "test panic in handler" {
			t.Errorf("Expected panic value 'test panic in handler', got %v", info.Value)
		}
		if info.Context["type"] != "http_request" {
			t.Errorf("Expected context type=http_request, got %v", info.Context["type"])
		}
		if info.Context["method"] != "POST" {
			t.Errorf("Expected context method=POST, got %v", info.Context["method"])
		}
		if info.Context["path"] != "/panic-test" {
			t.Errorf("Expected context path=/panic-test, got %v", info.Context["path"])
		}
		recovered <- true
	}

	// Wrap with recovery middleware
	wrapped := recovery.HTTPMiddleware(handler, customHandler)

	// Create test request
	req := httptest.NewRequest("POST", "/panic-test", nil)
	rr := httptest.NewRecorder()

	// Execute request (should not panic)
	wrapped.ServeHTTP(rr, req)

	// Verify response is 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}

	// Verify recovery handler was called
	select {
	case <-recovered:
		// Test passed
	default:
		t.Error("Recovery handler was not called")
	}
}

func TestHTTPMiddleware_WithPanic_DefaultHandler(t *testing.T) {
	// Create a handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test default handler")
	})

	// Use nil handler (should use DefaultHandler)
	wrapped := recovery.HTTPMiddleware(handler, nil)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Execute request (should not panic)
	wrapped.ServeHTTP(rr, req)

	// Verify response is 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}

// ==================== HandlerFuncMiddleware Tests ====================

func TestHandlerFuncMiddleware_NoPanic(t *testing.T) {
	// Create a simple handler that doesn't panic
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("HandlerFunc OK"))
	}

	// Wrap with recovery middleware
	wrapped := recovery.HandlerFuncMiddleware(handler, nil)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Execute request
	wrapped(rr, req)

	// Verify response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	if rr.Body.String() != "HandlerFunc OK" {
		t.Errorf("Expected body 'HandlerFunc OK', got '%s'", rr.Body.String())
	}
}

func TestHandlerFuncMiddleware_WithPanic(t *testing.T) {
	recovered := make(chan bool, 1)

	// Create a handler that panics
	handler := func(w http.ResponseWriter, r *http.Request) {
		panic("handler func panic")
	}

	// Custom recovery handler
	customHandler := func(info recovery.PanicInfo) {
		if info.Value != "handler func panic" {
			t.Errorf("Expected panic value 'handler func panic', got %v", info.Value)
		}
		if info.Context["method"] != "DELETE" {
			t.Errorf("Expected context method=DELETE, got %v", info.Context["method"])
		}
		recovered <- true
	}

	// Wrap with recovery middleware
	wrapped := recovery.HandlerFuncMiddleware(handler, customHandler)

	// Create test request
	req := httptest.NewRequest("DELETE", "/delete-test", nil)
	rr := httptest.NewRecorder()

	// Execute request (should not panic)
	wrapped(rr, req)

	// Verify response is 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}

	// Verify recovery handler was called
	select {
	case <-recovered:
		// Test passed
	default:
		t.Error("Recovery handler was not called")
	}
}

func TestHandlerFuncMiddleware_CapturesAllContext(t *testing.T) {
	recovered := make(chan recovery.PanicInfo, 1)

	handler := func(w http.ResponseWriter, r *http.Request) {
		panic("context test")
	}

	customHandler := func(info recovery.PanicInfo) {
		recovered <- info
	}

	wrapped := recovery.HandlerFuncMiddleware(handler, customHandler)

	// Create request with remote addr
	req := httptest.NewRequest("PUT", "/context/path?query=value", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()

	wrapped(rr, req)

	select {
	case info := <-recovered:
		// Verify all context fields
		if info.Context["type"] != "http_request" {
			t.Error("Missing or wrong type in context")
		}
		if info.Context["method"] != "PUT" {
			t.Error("Missing or wrong method in context")
		}
		if info.Context["path"] != "/context/path" {
			t.Error("Missing or wrong path in context")
		}
		if info.Context["remote"] != "192.168.1.1:12345" {
			t.Error("Missing or wrong remote in context")
		}
		if info.StackTrace == "" {
			t.Error("Missing stack trace")
		}
		if info.Timestamp.IsZero() {
			t.Error("Timestamp should not be zero")
		}
	default:
		t.Error("Recovery handler was not called")
	}
}
