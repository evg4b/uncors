package tui_test

import (
	"testing"

	"github.com/evg4b/uncors/testing/testutils"

	"github.com/evg4b/uncors/internal/tui"
	"github.com/stretchr/testify/assert"
)

var expectedLogo = []byte{
	0x1b, 0x5b, 0x33, 0x38, 0x3b, 0x32, 0x3b, 0x32, 0x32, 0x30, 0x3b, 0x31, 0x3b, 0x30, 0x6d, 0xe2, 0x96, 0x88,
	0xe2, 0x96, 0x88, 0x20, 0x20, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0xe2, 0x96, 0x88, 0xe2,
	0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x1b, 0x5b,
	0x30, 0x6d, 0x1b, 0x5b, 0x33, 0x38, 0x3b, 0x32, 0x3b, 0x32, 0x35, 0x35, 0x3b, 0x32, 0x31, 0x31, 0x3b, 0x30,
	0x6d, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2,
	0x96, 0x88, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96,
	0x88, 0xe2, 0x96, 0x88, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88,
	0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2,
	0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x1b, 0x5b, 0x30, 0x6d, 0xa, 0x1b, 0x5b,
	0x33, 0x38, 0x3b, 0x32, 0x3b, 0x32, 0x32, 0x30, 0x3b, 0x31, 0x3b, 0x30, 0x6d, 0xe2, 0x96, 0x88, 0xe2, 0x96,
	0x88, 0x20, 0x20, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88,
	0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x1b, 0x5b,
	0x30, 0x6d, 0x1b, 0x5b, 0x33, 0x38, 0x3b, 0x32, 0x3b, 0x32, 0x35, 0x35, 0x3b, 0x32, 0x31, 0x31, 0x3b, 0x30,
	0x6d, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96,
	0x88, 0x20, 0x20, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88,
	0x20, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20,
	0x20, 0x20, 0x20, 0x1b, 0x5b, 0x30, 0x6d, 0xa, 0x1b, 0x5b, 0x33, 0x38, 0x3b, 0x32, 0x3b, 0x32, 0x32, 0x30,
	0x3b, 0x31, 0x3b, 0x30, 0x6d, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20, 0x20, 0x20, 0xe2, 0x96, 0x88,
	0xe2, 0x96, 0x88, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20,
	0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x1b, 0x5b, 0x30, 0x6d, 0x1b, 0x5b, 0x33, 0x38, 0x3b, 0x32,
	0x3b, 0x32, 0x35, 0x35, 0x3b, 0x32, 0x31, 0x31, 0x3b, 0x30, 0x6d, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20,
	0x20, 0x20, 0x20, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20, 0x20, 0x20, 0xe2, 0x96, 0x88,
	0xe2, 0x96, 0x88, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96,
	0x88, 0xe2, 0x96, 0x88, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88,
	0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x1b, 0x5b, 0x30, 0x6d, 0xa, 0x1b, 0x5b, 0x33, 0x38,
	0x3b, 0x32, 0x3b, 0x32, 0x32, 0x30, 0x3b, 0x31, 0x3b, 0x30, 0x6d, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20,
	0x20, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20,
	0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x1b, 0x5b, 0x30, 0x6d,
	0x1b, 0x5b, 0x33, 0x38, 0x3b, 0x32, 0x3b, 0x32, 0x35, 0x35, 0x3b, 0x32, 0x31, 0x31, 0x3b, 0x30, 0x6d, 0xe2,
	0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20,
	0x20, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20,
	0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96,
	0x88, 0x1b, 0x5b, 0x30, 0x6d, 0xa, 0x1b, 0x5b, 0x33, 0x38, 0x3b, 0x32, 0x3b, 0x32, 0x32, 0x30, 0x3b, 0x31,
	0x3b, 0x30, 0x6d, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96,
	0x88, 0xe2, 0x96, 0x88, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20, 0x20, 0xe2, 0x96, 0x88,
	0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x1b, 0x5b, 0x30, 0x6d, 0x1b, 0x5b, 0x33, 0x38,
	0x3b, 0x32, 0x3b, 0x32, 0x35, 0x35, 0x3b, 0x32, 0x31, 0x31, 0x3b, 0x30, 0x6d, 0x20, 0xe2, 0x96, 0x88, 0xe2,
	0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20, 0xe2, 0x96,
	0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20,
	0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0x20, 0x20, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0x20, 0xe2, 0x96,
	0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96, 0x88, 0xe2, 0x96,
	0x88, 0x1b, 0x5b, 0x30, 0x6d, 0xa, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
	0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
	0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x3a, 0x20, 0x30, 0x2e,
	0x31, 0x2e, 0x30,
}

func TestLogo(t *testing.T) {
	t.Run("Logo", testutils.WithTrueColor(func(t *testing.T) {
		version := "0.1.0"
		logo := tui.Logo(version)
		assert.Equal(t, expectedLogo, []byte(logo))
	}))
}
