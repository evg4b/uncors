//go:build release

package infrastructure

func PanicInterceptor(action func(any)) {
	if recovered := recover(); recovered != nil {
		action(recovered)
	}
}
