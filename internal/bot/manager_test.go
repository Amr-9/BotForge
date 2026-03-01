package bot

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Amr-9/botforge/internal/recovery"
)

// ==================== NewManager Tests ====================

func TestNewManager_Initialization(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.webhookURL != "https://example.com" {
		t.Errorf("Expected webhookURL 'https://example.com', got '%s'", m.webhookURL)
	}
	if m.bots == nil {
		t.Error("bots map should be initialized")
	}
	if m.botIDs == nil {
		t.Error("botIDs map should be initialized")
	}
	if m.restartPolicies == nil {
		t.Error("restartPolicies map should be initialized")
	}
	if m.restartControllers == nil {
		t.Error("restartControllers map should be initialized")
	}
	if m.preloadCancels == nil {
		t.Error("preloadCancels map should be initialized")
	}
}

func TestNewManager_EmptyWebhookURL(t *testing.T) {
	m := NewManager(nil, nil, "")

	if m.webhookURL != "" {
		t.Errorf("Expected empty webhookURL, got '%s'", m.webhookURL)
	}
}

func TestNewManagerWithRecovery_SetsHandler(t *testing.T) {
	handlerCalled := false
	customHandler := func(info recovery.PanicInfo) {
		handlerCalled = true
	}

	m := NewManagerWithRecovery(nil, nil, "https://example.com", customHandler)

	if m == nil {
		t.Fatal("NewManagerWithRecovery returned nil")
	}
	if m.recoveryHandler == nil {
		t.Error("recoveryHandler should be set")
	}

	// Verify handler is called on panic
	func() {
		defer recovery.Recover(m.recoveryHandler, nil)
		panic("test panic")
	}()

	if !handlerCalled {
		t.Error("Custom recovery handler should have been called")
	}
}

// ==================== IsRunning Tests ====================

func TestIsRunning_EmptyManager(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	if m.IsRunning("anytoken12345678") {
		t.Error("IsRunning should return false for empty manager")
	}
}

func TestIsRunning_BotInMap(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	m.mu.Lock()
	m.bots["existingtoken1234"] = nil
	m.mu.Unlock()

	if !m.IsRunning("existingtoken1234") {
		t.Error("IsRunning should return true when token is in map")
	}
}

func TestIsRunning_DifferentToken(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	m.mu.Lock()
	m.bots["tokenAAAAAAAAAAAA"] = nil
	m.mu.Unlock()

	if m.IsRunning("tokenBBBBBBBBBBBB") {
		t.Error("IsRunning should return false for a different token")
	}
}

func TestIsRunning_AfterRemoval(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	m.mu.Lock()
	m.bots["removetoken12345"] = nil
	m.mu.Unlock()

	m.mu.Lock()
	delete(m.bots, "removetoken12345")
	m.mu.Unlock()

	if m.IsRunning("removetoken12345") {
		t.Error("IsRunning should return false after removal")
	}
}

// ==================== GetRunningCount Tests ====================

func TestGetRunningCount_Empty(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	if count := m.GetRunningCount(); count != 0 {
		t.Errorf("Expected 0 for empty manager, got %d", count)
	}
}

func TestGetRunningCount_OneBot(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	m.mu.Lock()
	m.bots["singletoken12345"] = nil
	m.mu.Unlock()

	if count := m.GetRunningCount(); count != 1 {
		t.Errorf("Expected 1, got %d", count)
	}
}

func TestGetRunningCount_MultipleBots(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	m.mu.Lock()
	m.bots["token1234567890aa"] = nil
	m.bots["token0987654321bb"] = nil
	m.bots["tokenaabbccddeeff"] = nil
	m.mu.Unlock()

	if count := m.GetRunningCount(); count != 3 {
		t.Errorf("Expected 3, got %d", count)
	}
}

// ==================== GetBotByID Tests ====================

func TestGetBotByID_EmptyManager(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	_, _, err := m.GetBotByID(999)
	if err == nil {
		t.Error("Expected error for non-existent bot ID in empty manager")
	}
}

func TestGetBotByID_IDNotRegistered(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	m.mu.Lock()
	m.bots["sometoken1234567"] = nil
	m.botIDs["sometoken1234567"] = 10
	m.mu.Unlock()

	_, _, err := m.GetBotByID(999)
	if err == nil {
		t.Error("Expected error for ID that is not in botIDs map")
	}
}

func TestGetBotByID_Found(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	token := "foundtoken1234567"
	m.mu.Lock()
	m.bots[token] = nil
	m.botIDs[token] = 42
	m.mu.Unlock()

	_, gotToken, err := m.GetBotByID(42)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if gotToken != token {
		t.Errorf("Expected token '%s', got '%s'", token, gotToken)
	}
}

func TestGetBotByID_IDExistsButBotMissing(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	// Register ID without corresponding bot entry (inconsistent state)
	m.mu.Lock()
	m.botIDs["orphantoken12345"] = 77
	m.mu.Unlock()

	_, _, err := m.GetBotByID(77)
	if err == nil {
		t.Error("Expected error when bot ID exists but bot is missing from bots map")
	}
}

func TestGetBotByID_MultipleBotsCorrectLookup(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	m.mu.Lock()
	m.bots["tokenone111111111"] = nil
	m.botIDs["tokenone111111111"] = 1
	m.bots["tokentwo222222222"] = nil
	m.botIDs["tokentwo222222222"] = 2
	m.bots["tokenthr333333333"] = nil
	m.botIDs["tokenthr333333333"] = 3
	m.mu.Unlock()

	_, gotToken, err := m.GetBotByID(2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if gotToken != "tokentwo222222222" {
		t.Errorf("Expected 'tokentwo222222222', got '%s'", gotToken)
	}
}

// ==================== StopBot Tests ====================

func TestStopBot_NonExistentToken(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	// Should not panic
	m.StopBot("doesnotexist1234")

	if m.GetRunningCount() != 0 {
		t.Error("Count should remain 0")
	}
}

func TestStopBot_RemovesFromBotsMap(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")
	token := "stopbottoken12345"

	m.mu.Lock()
	m.bots[token] = nil
	m.botIDs[token] = 1
	m.restartPolicies[token] = recovery.NewRestartPolicy(3, time.Second, time.Minute)
	m.restartControllers[token] = recovery.NewRestartController()
	m.preloadCancels[token] = func() {}
	m.mu.Unlock()

	m.StopBot(token)
	time.Sleep(50 * time.Millisecond) // let SafeGo goroutine finish

	if m.IsRunning(token) {
		t.Error("Bot should be removed from bots map after StopBot")
	}
	m.mu.RLock()
	_, hasID := m.botIDs[token]
	_, hasPolicy := m.restartPolicies[token]
	_, hasController := m.restartControllers[token]
	_, hasCancel := m.preloadCancels[token]
	m.mu.RUnlock()

	if hasID {
		t.Error("botID entry should be removed")
	}
	if hasPolicy {
		t.Error("restartPolicy entry should be removed")
	}
	if hasController {
		t.Error("restartController entry should be removed")
	}
	if hasCancel {
		t.Error("preloadCancel entry should be removed")
	}
}

func TestStopBot_CallsPreloadCancel(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")
	token := "canceltoken123456"

	cancelCalled := false
	m.mu.Lock()
	m.bots[token] = nil
	m.botIDs[token] = 0
	m.restartControllers[token] = recovery.NewRestartController()
	m.preloadCancels[token] = func() { cancelCalled = true }
	m.mu.Unlock()

	m.StopBot(token)

	if !cancelCalled {
		t.Error("Preload cancel function should have been called by StopBot")
	}
}

func TestStopBot_IdempotentOnDoubleStop(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")
	token := "idempotenttoken12"

	m.mu.Lock()
	m.bots[token] = nil
	m.botIDs[token] = 0
	m.restartControllers[token] = recovery.NewRestartController()
	m.preloadCancels[token] = func() {}
	m.mu.Unlock()

	m.StopBot(token)
	// Second stop should be a no-op, not panic
	m.StopBot(token)

	if m.IsRunning(token) {
		t.Error("Bot should not be running after double stop")
	}
}

// ==================== StopAll Tests ====================

func TestStopAll_EmptyManager(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	// Should not panic
	m.StopAll()

	if m.GetRunningCount() != 0 {
		t.Error("Count should be 0 after StopAll on empty manager")
	}
}

func TestStopAll_RemovesAllBots(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	tokens := []string{"token111111111111", "token222222222222", "token333333333333"}
	m.mu.Lock()
	for _, token := range tokens {
		m.bots[token] = nil
		m.botIDs[token] = 0
		m.restartControllers[token] = recovery.NewRestartController()
		m.preloadCancels[token] = func() {}
	}
	m.mu.Unlock()

	m.StopAll()
	time.Sleep(50 * time.Millisecond)

	if count := m.GetRunningCount(); count != 0 {
		t.Errorf("Expected 0 bots after StopAll, got %d", count)
	}
}

func TestStopAll_CallsAllPreloadCancels(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	cancelCount := 0
	var cancelMu sync.Mutex

	tokens := []string{"token111111111111", "token222222222222"}
	m.mu.Lock()
	for _, token := range tokens {
		m.bots[token] = nil
		m.botIDs[token] = 0
		m.restartControllers[token] = recovery.NewRestartController()
		m.preloadCancels[token] = func() {
			cancelMu.Lock()
			cancelCount++
			cancelMu.Unlock()
		}
	}
	m.mu.Unlock()

	m.StopAll()

	cancelMu.Lock()
	got := cancelCount
	cancelMu.Unlock()

	if got != 2 {
		t.Errorf("Expected 2 cancel calls, got %d", got)
	}
}

// ==================== ServeHTTP Tests ====================

func TestServeHTTP_PathTooShort(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString("{}"))
	rr := httptest.NewRecorder()

	m.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for path with < 3 parts, got %d", rr.Code)
	}
}

func TestServeHTTP_EmptyToken(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	req := httptest.NewRequest(http.MethodPost, "/webhook/", bytes.NewBufferString("{}"))
	rr := httptest.NewRecorder()

	m.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for empty token, got %d", rr.Code)
	}
}

func TestServeHTTP_BotNotFound(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	req := httptest.NewRequest(http.MethodPost, "/webhook/unknowntoken123", bytes.NewBufferString("{}"))
	rr := httptest.NewRecorder()

	m.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for unknown token, got %d", rr.Code)
	}
}

func TestServeHTTP_InvalidJSON(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")
	token := "jsonerrortoken1234"

	m.mu.Lock()
	m.bots[token] = nil
	m.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/webhook/"+token, bytes.NewBufferString("not valid json {{{"))
	rr := httptest.NewRecorder()

	m.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestServeHTTP_ValidRequest_Returns200(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")
	token := "validtoken12345678"

	// Inject nil bot — ProcessUpdate will panic, but recovery catches it
	m.mu.Lock()
	m.bots[token] = nil
	m.mu.Unlock()

	body := `{"update_id": 1, "message": {"message_id": 1, "chat": {"id": 123}}}`
	req := httptest.NewRequest(http.MethodPost, "/webhook/"+token, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	m.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for valid request with recovery, got %d", rr.Code)
	}
}

func TestServeHTTP_EmptyBody(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")
	token := "emptybodytoken1234"

	m.mu.Lock()
	m.bots[token] = nil
	m.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/webhook/"+token, bytes.NewBufferString(""))
	rr := httptest.NewRecorder()

	m.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for empty body, got %d", rr.Code)
	}
}

// ==================== ManualPoller Tests ====================

func TestManualPoller_BlocksUntilStop(t *testing.T) {
	poller := &ManualPoller{}
	stop := make(chan struct{})
	done := make(chan struct{})

	go func() {
		poller.Poll(nil, nil, stop)
		close(done)
	}()

	// Verify it is blocking
	select {
	case <-done:
		t.Fatal("ManualPoller should be blocking, not stopped yet")
	case <-time.After(20 * time.Millisecond):
		// Expected: still blocking
	}

	// Close stop channel to unblock
	close(stop)

	select {
	case <-done:
		// Expected: stopped
	case <-time.After(200 * time.Millisecond):
		t.Error("ManualPoller should have unblocked after stop channel was closed")
	}
}

func TestManualPoller_StopsImmediatelyIfAlreadyClosed(t *testing.T) {
	poller := &ManualPoller{}
	stop := make(chan struct{})
	close(stop) // Already closed

	done := make(chan struct{})
	go func() {
		poller.Poll(nil, nil, stop)
		close(done)
	}()

	select {
	case <-done:
		// Expected: returns immediately
	case <-time.After(100 * time.Millisecond):
		t.Error("ManualPoller should return immediately when stop is already closed")
	}
}

// ==================== Concurrency Tests ====================

func TestManager_ConcurrentIsRunning(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")
	token := "concurrenttoken123"

	m.mu.Lock()
	m.bots[token] = nil
	m.mu.Unlock()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.IsRunning(token)
		}()
	}
	wg.Wait()
	// No data race = test passes (run with -race flag)
}

func TestManager_ConcurrentGetRunningCount(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	m.mu.Lock()
	m.bots["token111111111111"] = nil
	m.bots["token222222222222"] = nil
	m.mu.Unlock()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.GetRunningCount()
		}()
	}
	wg.Wait()
}

func TestManager_ConcurrentGetBotByID(t *testing.T) {
	m := NewManager(nil, nil, "https://example.com")

	m.mu.Lock()
	m.bots["concurrtoken12345"] = nil
	m.botIDs["concurrtoken12345"] = 55
	m.mu.Unlock()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.GetBotByID(55)
		}()
	}
	wg.Wait()
}
