package logger

import (
	"fmt"
	"os"
	"sync"
)

// Logger defines the interface for MCP logging
type Logger interface {
	// Logf logs a formatted message
	Logf(format string, args ...interface{})
}

// NoopLogger implements Logger with no-op operations
type NoopLogger struct{}

func NewNoopLogger() *NoopLogger {
	return &NoopLogger{}
}

func (l *NoopLogger) Logf(format string, args ...interface{}) {}

// StderrLogger implements Logger using stderr
type StderrLogger struct {
	prefix string
	mu     sync.Mutex
}

func NewStderrLogger(prefix string) *StderrLogger {
	return &StderrLogger{prefix: prefix}
}

func (l *StderrLogger) Logf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	fmt.Fprintf(os.Stderr, "[%s] ", l.prefix)
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
