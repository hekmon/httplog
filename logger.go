package httplog

import (
	"context"
	"log/slog"
	"sync/atomic"
)

var (
	// DefaultBodyMaxRead is the default maximum number of bytes a body must have to be logged.
	// Its value is copied in the New() constructor.
	DefaultBodyMaxRead int64 = 10000
	// DefaultSanitizeHeaders is the default list of headers to sanitize in the debug log.
	SanitizeHeaders = []string{"Authorization"}
)

// Logger is a HTTP request/response logging utility.
// It wraps a slog.Logger and provides additional functionality for logging HTTP requests and responses.
// Instanciate with New().
type Logger struct {
	slogger     *slog.Logger
	requests    atomic.Uint64
	bodyMaxRead int64
}

// New creates a new HTTP request/response logging utility.
func New(logger *slog.Logger) (l *Logger) {
	if logger == nil {
		return
	}
	return &Logger{
		slogger:     logger,
		bodyMaxRead: DefaultBodyMaxRead,
	}
}

// TotalRequests returns the number of requests that went thru the logger.
// Current, yet unfulfilled, requests are also taking into account.
func (l *Logger) TotalRequests() uint64 {
	return l.requests.Load()
}

func GetReqIDSLogAttr(reqCtx context.Context) slog.Attr {
	// Retreive the request ID from context inserted by the log middleware and return it as a slog.Attr
	return slog.Uint64(ReqIDKeyName, reqCtx.Value(ReqIDKey).(uint64))
}
