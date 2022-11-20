//go:build release

package infrastructure

func PanicInterceptor(action func(interface{})) {
	if recovered := recover(); recovered != nil {
		action(recovered)
	}
}
