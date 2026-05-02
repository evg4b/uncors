package testutils

import (
	"net"
	"net/url"
	"testing"

	"github.com/evg4b/uncors/testing/hosts"
	"github.com/stretchr/testify/require"
)

func GetFreePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp4", hosts.Loopback.Port(0)) //nolint:noctx
	require.NoError(t, err)
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)

	return addr.Port
}

func GetFreePorts(t *testing.T, count int) []int {
	t.Helper()

	ports := make([]int, 0, count)
	for i := 0; i < count; i++ {
		ports = append(ports, GetFreePort(t))
	}

	return ports
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
