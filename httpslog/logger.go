package httpslog

import (
	"log/slog"
	"sync/atomic"
)

const (
	// DefaultBodyMaxRead is the default maximum number of bytes a body must have to be logged.
	DefaultBodyMaxRead int64 = 10000
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
func New(logger *slog.Logger, debugBodyMaxRead int64) (l *Logger) {
	if logger == nil {
		return
	}
	if debugBodyMaxRead < 0 {
		debugBodyMaxRead = DefaultBodyMaxRead
	}
	return &Logger{
		slogger:     logger,
		bodyMaxRead: debugBodyMaxRead,
	}
}
