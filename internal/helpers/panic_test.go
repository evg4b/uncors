//go:build release

package helpers_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/stretchr/testify/assert"
)

func TestPanicInterceptorRecovers(t *testing.T) {
	var recovered any

	assert.NotPanics(t, func() {
		defer helpers.PanicInterceptor(func(data any) {
			recovered = data
		})

		panic(testconstants.ErrTest1)
	})

	assert.Equal(t, testconstants.ErrTest1, recovered)
}
