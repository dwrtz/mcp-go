package testutil

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

// TestLogger provides a thread-safe logging buffer for tests
type TestLogger struct {
	t   *testing.T
	buf *bytes.Buffer
	mu  sync.Mutex
}

// NewTestLogger creates a new TestLogger
func NewTestLogger(t *testing.T) *TestLogger {
	return &TestLogger{
		t:   t,
		buf: &bytes.Buffer{},
	}
}

// Logf writes a formatted log message to the buffer
func (l *TestLogger) Logf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	msg := fmt.Sprintf(format+"\n", args...)
	l.buf.WriteString(msg)
	l.t.Log(msg) // Also write to test log
}

// String returns the current contents of the log buffer
func (l *TestLogger) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}
