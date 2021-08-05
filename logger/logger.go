package logger

// Field represents a structured field to be added to a Log entry.
type Field struct {
	Key   string
	Value interface{}
}

// Logger is a structured logger capable of printing information about
// the execution of a component at various levels.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Error(msg string, fields ...Field)
}
