package httplog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/hekmon/httplog/catcherflusher"
)

// ReqIDType is a custom type for storing a unique ID for each HTTP request within the request context.
type ReqIDType string

const (
	// ReqIDKey is a reference key to store a unique ID for each HTTP request within the context.
	// Use it to retreive the unique ID of a HTTP request in the wrapped handler from the request context.
	ReqIDKey ReqIDType = "reqid"
	// ReqIDKeyName is a reference slog key name that you can use to be consistent on how the key should be name.
	ReqIDKeyName string = "request_id"
)

// Log is a HTTP middleware that logs HTTP requests and responses. Use it to decorates your actual http handlers.
// Request body and response body are logged only if the wrapped slogger's level is set to LevelDebug or lower.
func (l *Logger) Log(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Generate a uniq ID for this request
		reqID := l.requests.Add(1)
		logger := l.slogger.With(slog.Uint64(ReqIDKeyName, reqID))
		// Log the request
		if logger.Handler().Enabled(r.Context(), slog.LevelInfo) {
			logger.InfoContext(r.Context(), "HTTP request received",
				slog.String("host", r.Host),
				slog.String("method", r.Method),
				slog.String("URI", r.URL.RequestURI()),
				slog.String("client", r.RemoteAddr),
				slog.Any("headers", r.Header),
			)
		}
		// If debug is on, try to log body up to a certain size
		if r.ContentLength > 0 && logger.Handler().Enabled(r.Context(), slog.LevelDebug) {
			var bodyAttribute slog.Attr
			if r.ContentLength <= l.bodyMaxRead {
				// Read body
				var bodyBuffer bytes.Buffer
				if _, err := io.CopyN(&bodyBuffer, r.Body, l.bodyMaxRead); err != nil && err != io.EOF {
					slog.ErrorContext(r.Context(), "Failed to read body",
						slog.String("host", r.Host),
						slog.String("method", r.Method),
						slog.String("URI", r.URL.RequestURI()),
						slog.String("client", r.RemoteAddr),
						slog.Any("headers", r.Header),
						slog.String("error", err.Error()),
					)
					http.Error(
						w,
						fmt.Sprintf("%s: failed to read body: %s",
							http.StatusText(http.StatusInternalServerError),
							err.Error(),
						),
						http.StatusInternalServerError,
					)
					return
				}
				// Add body content to the future log
				bodyAttribute = slog.String("body", bodyBuffer.String())
				// Make the body available again
				r.Body = io.NopCloser(&bodyBuffer)
			} else {
				bodyAttribute = slog.String("skipped", fmt.Sprintf("body exceeds max debug size of %d", l.bodyMaxRead))
			}
			logger.DebugContext(r.Context(), "HTTP request body",
				bodyAttribute,
			)
		}
		// Pass to the wrapped handler
		flusherCatcher := catcherflusher.NewResponseWriter(w, logger.Handler().Enabled(r.Context(), slog.LevelDebug))
		next.ServeHTTP(
			flusherCatcher,
			r.WithContext(context.WithValue(r.Context(), ReqIDKey, reqID)),
		)
		// Log the response
		logger.InfoContext(r.Context(), "HTTP request handled",
			slog.Int("response_code", flusherCatcher.GetResponseCode()),
			slog.String("response_status", http.StatusText(flusherCatcher.GetResponseCode())),
			slog.Duration("response_time", time.Since(start)),
		)
		if logger.Handler().Enabled(r.Context(), slog.LevelDebug) {
			body := flusherCatcher.GetBody()
			if int64(len(body)) <= l.bodyMaxRead {
				logger.DebugContext(r.Context(), "HTTP response",
					slog.String("response_body", string(body)),
					slog.Int("response_size", len(body)),
				)
			} else {
				logger.DebugContext(r.Context(), "HTTP response",
					slog.String("skipped", fmt.Sprintf("body exceeds max debug size of %d", l.bodyMaxRead)),
					slog.Int("response_size", len(body)),
				)
			}
		}
	})
}
