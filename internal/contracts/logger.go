package contracts

type Logger interface {
	Error(a ...any)
	Errorf(template string, a ...any)
	Warning(a ...any)
	Warningf(template string, a ...any)
	Info(a ...any)
	Infof(template string, a ...any)
	Debug(a ...any)
	Debugf(template string, a ...any)
	PrintResponse(request *Request, code int)
}
