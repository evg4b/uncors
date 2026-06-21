package testutils

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func Close(t *testing.T, closer io.Closer) {
	err := closer.Close()
	require.NoError(t, err)
}
