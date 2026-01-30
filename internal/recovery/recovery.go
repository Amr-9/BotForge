package recovery

import (
	"log"
	"runtime/debug"
	"sync"
	"time"
)

// PanicInfo holds information about a recovered panic
type PanicInfo struct {
	Timestamp  time.Time
	Value      interface{}
	StackTrace string
	Context    map[string]string
}

// Handler is a function called when a panic is recovered
type Handler func(info PanicInfo)

// DefaultHandler logs panic information with full stack trace
func DefaultHandler(info PanicInfo) {
	log.Printf("[PANIC RECOVERED] Time: %s\nContext: %v\nValue: %v\nStack:\n%s",
		info.Timestamp.Format(time.RFC3339),
		info.Context,
		info.Value,
		info.StackTrace)
}

// Recover captures panic information and calls the handler.
// Use with defer: defer recovery.Recover(handler, context)
func Recover(handler Handler, context map[string]string) {
	if r := recover(); r != nil {
		info := PanicInfo{
			Timestamp:  time.Now(),
			Value:      r,
			StackTrace: string(debug.Stack()),
			Context:    context,
		}
		if handler != nil {
			handler(info)
		} else {
			DefaultHandler(info)
		}
	}
}

// RestartPolicy controls restart behavior after panic with exponential backoff
type RestartPolicy struct {
	MaxRetries     int
	BaseDelay      time.Duration
	MaxDelay       time.Duration
	currentRetries int
	mu             sync.Mutex
}

// NewRestartPolicy creates a policy with exponential backoff
func NewRestartPolicy(maxRetries int, baseDelay, maxDelay time.Duration) *RestartPolicy {
	return &RestartPolicy{
		MaxRetries: maxRetries,
		BaseDelay:  baseDelay,
		MaxDelay:   maxDelay,
	}
}

// ShouldRestart returns true if restart is allowed, with delay duration
func (p *RestartPolicy) ShouldRestart() (bool, time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentRetries >= p.MaxRetries {
		return false, 0
	}

	// Calculate delay with exponential backoff: baseDelay * 2^retries
	delay := p.BaseDelay * time.Duration(1<<p.currentRetries)
	if delay > p.MaxDelay {
		delay = p.MaxDelay
	}

	p.currentRetries++
	return true, delay
}

// Reset resets the retry counter (call after successful startup period)
func (p *RestartPolicy) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentRetries = 0
}

// GetRetryCount returns current retry count
func (p *RestartPolicy) GetRetryCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentRetries
}
