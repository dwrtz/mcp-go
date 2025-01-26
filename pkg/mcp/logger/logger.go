package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger defines the interface for MCP logging
type Logger interface {
	// Logf logs a formatted message
	Logf(format string, args ...interface{})
	// Close closes the logger and frees resources
	Close() error
}

// FileLogger implements Logger using file-based logging
type FileLogger struct {
	mu       sync.Mutex
	file     *os.File
	filepath string
}

// NewFileLogger creates a new file logger
func NewFileLogger(logPath string) (*FileLogger, error) {
	// Create log directory if it doesn't exist
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file with append mode
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &FileLogger{
		file:     file,
		filepath: logPath,
	}, nil
}

// Logf implements Logger.Logf
func (l *FileLogger) Logf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Add timestamp to log message
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf("%s %s\n", timestamp, fmt.Sprintf(format, args...))

	// Write to file
	if _, err := l.file.WriteString(msg); err != nil {
		// If we can't write to the file, there's not much we can do
		// Maybe in the future we could implement some fallback mechanism
		return
	}
}

// Close implements Logger.Close
func (l *FileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		if err := l.file.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
		l.file = nil
	}
	return nil
}

// NoopLogger implements Logger with no-op operations
type NoopLogger struct{}

func NewNoopLogger() *NoopLogger {
	return &NoopLogger{}
}

func (l *NoopLogger) Logf(format string, args ...interface{}) {}
func (l *NoopLogger) Close() error                            { return nil }

// StderrLogger implements Logger using stderr
type StderrLogger struct {
	prefix string
}

func NewStderrLogger(prefix string) *StderrLogger {
	return &StderrLogger{prefix: prefix}
}

func (l *StderrLogger) Logf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[%s] ", l.prefix)
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
