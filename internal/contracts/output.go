package contracts

type Output interface {
	Info(msg any)
	Error(msg any)
	Errorf(msg string, args ...any)
}
