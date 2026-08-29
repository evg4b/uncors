//go:build release

package cli

func PanicInterceptor(action func(any)) {
	if recovered := recover(); recovered != nil {
		action(recovered)
	}
}
