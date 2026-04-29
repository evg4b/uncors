package contracts

import "io"

type Output interface {
	io.Writer
	Info(msg any)
	Error(msg any)
	Errorf(msg string, args ...any)

	Print(msg any)

	Request(data *ReqestData)
}
