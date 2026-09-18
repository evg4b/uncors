package testutils

import (
	"net"
	"testing"

	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/stretchr/testify/require"
)

func GetFreePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp4", hosts.Loopback.Port(0).String()) //nolint:noctx
	require.NoError(t, err)

	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)

	return addr.Port
}

func GetFreePorts(t *testing.T, count int) []int {
	t.Helper()

	ports := make([]int, 0, count)
	for range count {
		ports = append(ports, GetFreePort(t))
	}

	return ports
}

func IsPortFree(port int) bool {
	l, err := net.Listen("tcp", hosts.Loopback.Port(port).String()) // nolint: noctx
	if err != nil {
		return false
	}
	defer l.Close()

	return true
}

// OccupyPort binds port for the rest of the test, so whatever is under test
// cannot bind it and has to report the failure.
func OccupyPort(t *testing.T, port int) {
	t.Helper()

	listenConfig := &net.ListenConfig{}

	listener, err := listenConfig.Listen(t.Context(), "tcp4", hosts.Loopback.Port(port).String())
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })
}

func JoinPath(base string, elem ...string) string {
	joined, err := urlt.JoinPath(base, elem...)
	if err != nil {
		panic(err)
	}

	return joined
}
