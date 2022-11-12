package contracts

import (
	"net/http"
)

type Logger interface {
	Error(a ...interface{})
	Errorf(template string, a ...interface{})
	Warning(a ...interface{})
	Warningf(template string, a ...interface{})
	Info(a ...interface{})
	Infof(template string, a ...interface{})
	Debug(a ...interface{})
	Debugf(template string, a ...interface{})
	PrintResponse(response *http.Response)
}
