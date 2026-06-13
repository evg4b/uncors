//go:build integration

package tls_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/tests/integration/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTLS(t *testing.T) {
	env := harness.New(t, harness.WithBackendHandler(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "secure")
	}))

	t.Run("client trusting the proxy CA completes the handshake", func(t *testing.T) {
		request, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			env.Proxy.HTTPSURL("/"),
			nil,
		)
		require.NoError(t, err)

		response, err := env.Client.Do(request)
		require.NoError(t, err)

		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "secure", string(body))
		require.NotNil(t, response.TLS, "response must have travelled over TLS")
		assert.True(t, response.TLS.HandshakeComplete)
	})

	t.Run("client not trusting the proxy CA is rejected", func(t *testing.T) {
		// An empty pool trusts nothing, so verifying the proxy's leaf must fail.
		untrusting := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS13,
					RootCAs:    x509.NewCertPool(),
				},
			},
		}

		request, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			env.Proxy.HTTPSURL("/"),
			nil,
		)
		require.NoError(t, err)

		response, err := untrusting.Do(request) //nolint:bodyclose // request must fail before a body exists
		require.Error(t, err)
		require.Nil(t, response)

		var certErr x509.UnknownAuthorityError
		assert.ErrorAs(t, err, &certErr)
	})
}
