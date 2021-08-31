package logger

import "testing"

var _ Logger = Test{}

// Test is a logger.Logger implementation using testing.T instance.
type Test struct{ t *testing.T }

// NewTest returns a new logger using the provided testing.T instance.
func NewTest(t *testing.T) Test {
	return Test{t: t}
}

// Debug uses t.Logf to print a debug message.
func (t Test) Debug(msg string, fields ...Field) {
	t.t.Logf("[debug] %s {args: %+v}\n", msg, fields)
}

// Info uses t.Logf to print an info message.
func (t Test) Info(msg string, fields ...Field) {
	t.t.Logf("[info] %s {args: %+v}\n", msg, fields)
}

// Error uses t.Logf to print an error message.
func (t Test) Error(msg string, fields ...Field) {
	t.t.Logf("[error] %s {args: %+v}\n", msg, fields)
}
