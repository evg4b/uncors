package contracts

type Logger interface {
	Error(msg any, keyvals ...any)
	Errorf(template string, a ...any)
	Warn(msg any, keyvals ...any)
	Warnf(template string, a ...any)
	Info(msg any, keyvals ...any)
	Infof(template string, a ...any)
	Debug(msg any, keyvals ...any)
	Debugf(template string, a ...any)
}
