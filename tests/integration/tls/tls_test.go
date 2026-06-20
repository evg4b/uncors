//go:build integration

package tls_test

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTLS(t *testing.T) {
	backend := integration.NewBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, "secure")
		assert.NoError(t, err)
	})
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://tls.local"),
			To:   backend.AsHost(),
		}},
	})

	t.Run("client trusting the proxy CA completes the handshake", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("tls.local", "/")))
		defer result.Response.Body.Close()

		body, err := io.ReadAll(result.Response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "secure", string(body))
		require.NotNil(t, result.Response.TLS, "response must have travelled over TLS")
		assert.True(t, result.Response.TLS.HandshakeComplete)
	})

	t.Run("client not trusting the proxy CA is rejected", func(t *testing.T) {
		untrusting := &http.Client{
			Transport: &http.Transport{
				// Resolve mapped hosts to the proxy but trust no CA, so the
				// failure is the handshake, not DNS.
				DialContext: env.Hosts.DialContext,
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS13,
					RootCAs:    x509.NewCertPool(),
				},
			},
		}

		req := integration.NewRequest(t, http.MethodGet, env.URL("tls.local", "/"))

		resp, err := untrusting.Do(req) //nolint:bodyclose
		require.Error(t, err)
		require.Nil(t, resp)

		var certErr x509.UnknownAuthorityError
		assert.ErrorAs(t, err, &certErr)
	})
}
