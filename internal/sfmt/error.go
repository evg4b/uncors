package sfmt

import "fmt"

func Errorf(format string, payload ...any) error {
	return fmt.Errorf(format, payload...) // nolint:goerr113
}
