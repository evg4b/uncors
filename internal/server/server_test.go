package server_test

import (
	"net"
	"net/http"
	"strconv"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/server"
	"github.com/phayes/freeport"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	instance := server.New()

	freePorts, err := freeport.GetFreePorts(3)
	require.NoError(t, err)

	hosts := lo.Map(freePorts, func(port int, _ int) string {
		return net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	})

	mappings := lo.Map(hosts, func(host string, _ int) config.Mapping {
		return config.Mapping{
			From: host,
			To:   "https://github.com",
		}
	})

	instance.Start(t.Context(), config.Mappings(mappings).GroupByPort())

	defer func() {
		err := instance.Shutdown(t.Context())
		require.NoError(t, err)
	}()

	for _, host := range hosts {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+host, nil)
		require.NoError(t, err)

		response, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)
	}
}
