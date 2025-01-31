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

// FileLogger implements Logger using a file
type FileLogger struct {
	file   *os.File
	prefix string
	mu     sync.Mutex
}

// NewFileLogger creates a new FileLogger that writes to the specified file path
func NewFileLogger(filepath string, prefix string) (*FileLogger, error) {
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &FileLogger{
		file:   file,
		prefix: prefix,
	}, nil
}

func (l *FileLogger) Logf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	fmt.Fprintf(l.file, "[%s] ", l.prefix)
	fmt.Fprintf(l.file, format+"\n", args...)
}

// Close closes the underlying file
func (l *FileLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
