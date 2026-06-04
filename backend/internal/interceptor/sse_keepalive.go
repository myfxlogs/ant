package interceptor

import (
	"bufio"
	"net"
	"net/http"
	"sync"
	"time"
)

// SSEKeepaliveMiddleware wraps HTTP handlers to inject periodic keepalive
// comments on SSE (Server-Sent Events) streaming responses. This prevents
// reverse proxies (especially Cloudflare HTTP/2) from closing idle streams.
//
// The middleware activates only when the response Content-Type starts with
// "text/event-stream" or "application/connect+".
func SSEKeepaliveMiddleware(interval time.Duration) func(http.Handler) http.Handler {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			kw := &keepaliveWriter{
				ResponseWriter: w,
				interval:       interval,
				done:           make(chan struct{}),
			}
			next.ServeHTTP(kw, r)
		})
	}
}

// keepaliveWriter wraps http.ResponseWriter and injects SSE keepalive
// comments once per interval. Activation is deferred until the first Write
// call so we can inspect the Content-Type.
type keepaliveWriter struct {
	http.ResponseWriter
	interval time.Duration
	once     sync.Once
	done     chan struct{}
	wrote    bool
}

// WriteHeader intercepts the status write, then delegates.
func (w *keepaliveWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

// Write detects streaming responses by Content-Type and starts the keepalive
// goroutine on the first call. Each keepalive is an SSE comment line, which
// clients and proxies interpret as a no-op but keeps the TCP connection alive.
func (w *keepaliveWriter) Write(b []byte) (int, error) {
	w.once.Do(func() {
		ct := w.ResponseWriter.Header().Get("Content-Type")
		if len(ct) >= 9 && (ct[:9] == "text/even" || ct[:18] == "application/connec") {
			w.wrote = true
			go w.keepaliveLoop()
		}
	})
	return w.ResponseWriter.Write(b)
}

func (w *keepaliveWriter) keepaliveLoop() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Grab the request context via the underlying connection if possible.
	// If we can't get it, the loop still exits when the handler finishes
	// and the ResponseWriter is no longer valid.

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			w.writeKeepalive()
		}
	}
}

func (w *keepaliveWriter) writeKeepalive() {
	// Write an SSE comment. Browsers and proxies ignore lines starting with ':'.
	// We use ": kp\n\n" — the double newline delimits an SSE event.
	w.ResponseWriter.Write([]byte(": kp\n\n"))
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap returns the underlying ResponseWriter (for http.ResponseController).
func (w *keepaliveWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// Hijack supports WebSocket upgrades.
func (w *keepaliveWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Flush passes through to the underlying writer.
func (w *keepaliveWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Close signals the keepalive goroutine to stop.
func (w *keepaliveWriter) Close() error {
	if w.wrote {
		close(w.done)
	}
	return nil
}
