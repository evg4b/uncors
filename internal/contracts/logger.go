package contracts

import "net/http"

type Logger interface {
	Error(a ...any)
	Errorf(template string, a ...any)
	Warning(a ...any)
	Warningf(template string, a ...any)
	Info(a ...any)
	Infof(template string, a ...any)
	Debug(a ...any)
	Debugf(template string, a ...any)
	PrintResponse(response *http.Response)
}
