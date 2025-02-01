package logger

import (
	"fmt"
	"os"
	"path"
	"strings"
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
// If filepath is `app.log`, it will be named `app.{pid}.log`.
func NewFileLogger(filepath string, prefix string) (*FileLogger, error) {
	// Add process ID to the filename
	dir, filename := path.Split(filepath)
	ext := path.Ext(filename)
	basename := strings.TrimSuffix(filename, ext)
	pid := os.Getpid()
	newPath := path.Join(dir, fmt.Sprintf("%s.%d%s", basename, pid, ext))

	// Open file with append mode
	file, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
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
