package recovery

import (
	"net/http"
	"runtime/debug"
	"time"
)

// HTTPMiddleware wraps an HTTP handler with panic recovery.
// If a panic occurs, it logs the panic and returns 500 Internal Server Error.
func HTTPMiddleware(next http.Handler, handler Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				info := PanicInfo{
					Timestamp:  time.Now(),
					Value:      err,
					StackTrace: string(debug.Stack()),
					Context: map[string]string{
						"type":   "http_request",
						"method": r.Method,
						"path":   r.URL.Path,
						"remote": r.RemoteAddr,
					},
				}
				if handler != nil {
					handler(info)
				} else {
					DefaultHandler(info)
				}

				// Return 500 to client
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// HandlerFuncMiddleware wraps an http.HandlerFunc with panic recovery.
func HandlerFuncMiddleware(next http.HandlerFunc, handler Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				info := PanicInfo{
					Timestamp:  time.Now(),
					Value:      err,
					StackTrace: string(debug.Stack()),
					Context: map[string]string{
						"type":   "http_request",
						"method": r.Method,
						"path":   r.URL.Path,
						"remote": r.RemoteAddr,
					},
				}
				if handler != nil {
					handler(info)
				} else {
					DefaultHandler(info)
				}

				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next(w, r)
	}
}
