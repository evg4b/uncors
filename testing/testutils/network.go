package testutils

import (
	"net"
	"net/url"
	"testing"

	"github.com/evg4b/uncors/testing/hosts"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
)

func GetFreePort(t *testing.T) int {
	t.Helper()

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	return port
}

func GetFreePorts(t *testing.T, count int) []int {
	t.Helper()

	port, err := freeport.GetFreePorts(count)
	require.NoError(t, err)

	return port
}

func IsPortFree(port int) bool {
	l, err := net.Listen("tcp", hosts.Loopback.Port(port)) // nolint: noctx
	if err != nil {
		return false
	}
	defer l.Close()

	return true
}

func JoinPath(base string, elem ...string) string {
	joined, err := url.JoinPath(base, elem...)
	if err != nil {
		panic(err)
	}

	return joined
}
