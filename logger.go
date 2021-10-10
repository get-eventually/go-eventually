package eventually

type LoggerFunc func(format string, args ...interface{})

type Logger struct {
	Debugf LoggerFunc
	Infof  LoggerFunc
	Errorf LoggerFunc
}
