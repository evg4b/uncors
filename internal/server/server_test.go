package server_test

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/server"
	"github.com/phayes/freeport"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	const expectedContent = "Test"

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

	targets := lo.Map(config.Mappings(mappings).GroupByPort(), func(group config.PortGroup, _ int) server.Target {
		return server.Target{
			Port: group.Port,
			Handler: contracts.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := fmt.Fprint(w, expectedContent)
				require.NoError(t, err)
			}),
		}
	})

	instance.Start(t.Context(), targets)

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
		data, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, expectedContent, string(data))
	}
}
