//go:build release

package infra

func PanicInterceptor(action func(any)) {
	if recovered := recover(); recovered != nil {
		action(recovered)
	}
}
