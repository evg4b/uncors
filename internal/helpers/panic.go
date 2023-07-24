//go:build release

package helpers

func PanicInterceptor(action func(any)) {
	if recovered := recover(); recovered != nil {
		action(recovered)
	}
}
