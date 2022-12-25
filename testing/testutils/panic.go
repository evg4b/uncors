package testutils

import (
	"testing"
)

func PanicWith(t *testing.T, action func(), expect func(value any)) {
	t.Helper()

	defer func() {
		t.Helper()
		err := recover()
		expect(err)
	}()

	action()
}
