package logger

// Field represents a structured field to be added to a Log entry.
type Field struct {
	Key   string
	Value interface{}
}

// With is an helper function to add a field in a functional way.
func With(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Logger is a structured logger capable of printing information about
// the execution of a component at various levels.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Error(msg string, fields ...Field)
}

// Debug delegates the debug log call to the provided logger, if not nil.
func Debug(l Logger, msg string, fields ...Field) {
	if l != nil {
		l.Debug(msg, fields...)
	}
}

// Info delegates the info log call to the provided logger, if not nil.
func Info(l Logger, msg string, fields ...Field) {
	if l != nil {
		l.Info(msg, fields...)
	}
}

// Error delegates the error log call to the provided logger, if not nil.
func Error(l Logger, msg string, fields ...Field) {
	if l != nil {
		l.Error(msg, fields...)
	}
}
