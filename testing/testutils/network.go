package testutils

import (
	"context"
	"net"
	"testing"

	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/stretchr/testify/require"
)

func GetFreePort(t *testing.T) int {
	t.Helper()

	listener, err := new(net.ListenConfig).Listen(t.Context(), "tcp4", hosts.Loopback.Port(0).String())
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
	listener, err := new(net.ListenConfig).Listen(context.Background(), "tcp", hosts.Loopback.Port(port).String())
	if err != nil {
		return false
	}

	_ = listener.Close()

	return true
}

func JoinPath(base string, elem ...string) string {
	joined, err := urlt.JoinPath(base, elem...)
	if err != nil {
		panic(err)
	}

	return joined
}
