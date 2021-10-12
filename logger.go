package eventually

// LoggerFunc is the function that prints a formatted log line.
type LoggerFunc func(format string, args ...interface{})

// Logger is the standard interface used by the library to log things.
type Logger struct {
	Debugf LoggerFunc
	Infof  LoggerFunc
	Errorf LoggerFunc
}

func (l Logger) LogDebugf(fn func(LoggerFunc)) {
	if l.Debugf != nil {
		fn(l.Debugf)
	}
}

func (l Logger) LogInfof(fn func(LoggerFunc)) {
	if l.Infof != nil {
		fn(l.Infof)
	}
}

func (l Logger) LogErrorf(fn func(LoggerFunc)) {
	if l.Errorf != nil {
		fn(l.Errorf)
	}
}
