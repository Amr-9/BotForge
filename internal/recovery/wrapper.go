package recovery

import (
	"context"
	"log"
	"runtime/debug"
	"sync"
	"time"
)

// RestartController allows external control over restart loops
type RestartController struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
}

// NewRestartController creates a new controller for managing restart loops
func NewRestartController() *RestartController {
	ctx, cancel := context.WithCancel(context.Background())
	return &RestartController{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Stop signals the restart loop to stop
func (rc *RestartController) Stop() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.cancel()
}

// Context returns the controller's context
func (rc *RestartController) Context() context.Context {
	return rc.ctx
}

// SafeGo runs a function in a goroutine with panic recovery.
// If the function panics, the panic is recovered and logged.
func SafeGo(fn func(), context map[string]string, handler Handler) {
	go func() {
		defer Recover(handler, context)
		fn()
	}()
}

// SafeGoWithRestart runs a function with automatic restart on panic.
// It uses exponential backoff between restarts and stops after max retries.
// onMaxRetries is called when all restart attempts are exhausted.
func SafeGoWithRestart(fn func(), context map[string]string, handler Handler, policy *RestartPolicy, onMaxRetries func()) {
	go func() {
		for {
			// Run the function with panic recovery
			func() {
				defer func() {
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
				}()
				fn()
			}()

			// Function returned or panicked - check restart policy
			shouldRestart, delay := policy.ShouldRestart()
			if !shouldRestart {
				log.Printf("[RECOVERY] Max retries (%d) exhausted for context: %v", policy.MaxRetries, context)
				if onMaxRetries != nil {
					onMaxRetries()
				}
				return
			}

			log.Printf("[RECOVERY] Restarting in %v (attempt %d/%d, context: %v)",
				delay, policy.GetRetryCount(), policy.MaxRetries, context)
			time.Sleep(delay)
		}
	}()
}

// SafeGoWithRestartAndController is like SafeGoWithRestart but supports external cancellation.
// The controller can be used to stop the restart loop when the service is intentionally stopped.
func SafeGoWithRestartAndController(fn func(), context map[string]string, handler Handler, policy *RestartPolicy, controller *RestartController, onMaxRetries func()) {
	go func() {
		for {
			// Check if cancelled before starting
			select {
			case <-controller.Context().Done():
				log.Printf("[RECOVERY] Restart loop cancelled for context: %v", context)
				return
			default:
			}

			panicked := false
			// Run the function with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
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
				}()
				fn()
			}()

			// Check if cancelled after function returns
			select {
			case <-controller.Context().Done():
				log.Printf("[RECOVERY] Restart loop cancelled for context: %v", context)
				return
			default:
			}

			// Only restart on panic, not on normal return
			if !panicked {
				log.Printf("[RECOVERY] Function returned normally, not restarting: %v", context)
				return
			}

			// Function panicked - check restart policy
			shouldRestart, delay := policy.ShouldRestart()
			if !shouldRestart {
				log.Printf("[RECOVERY] Max retries (%d) exhausted for context: %v", policy.MaxRetries, context)
				if onMaxRetries != nil {
					onMaxRetries()
				}
				return
			}

			log.Printf("[RECOVERY] Restarting in %v (attempt %d/%d, context: %v)",
				delay, policy.GetRetryCount(), policy.MaxRetries, context)

			// Use select with timer to allow cancellation during sleep
			select {
			case <-controller.Context().Done():
				log.Printf("[RECOVERY] Restart cancelled during backoff for context: %v", context)
				return
			case <-time.After(delay):
			}
		}
	}()
}

// SafeGoWithRestartAndReset is like SafeGoWithRestart but resets the retry counter
// after the function runs successfully for the specified duration.
func SafeGoWithRestartAndReset(fn func(), context map[string]string, handler Handler, policy *RestartPolicy, successDuration time.Duration, onMaxRetries func()) {
	go func() {
		for {
			startTime := time.Now()
			panicked := false

			// Run the function with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
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
				}()
				fn()
			}()

			// If function ran successfully for long enough, reset retry counter
			if !panicked && time.Since(startTime) >= successDuration {
				policy.Reset()
			}

			// Function returned or panicked - check restart policy
			shouldRestart, delay := policy.ShouldRestart()
			if !shouldRestart {
				log.Printf("[RECOVERY] Max retries (%d) exhausted for context: %v", policy.MaxRetries, context)
				if onMaxRetries != nil {
					onMaxRetries()
				}
				return
			}

			log.Printf("[RECOVERY] Restarting in %v (attempt %d/%d, context: %v)",
				delay, policy.GetRetryCount(), policy.MaxRetries, context)
			time.Sleep(delay)
		}
	}()
}
