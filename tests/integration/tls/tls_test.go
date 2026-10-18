//go:build integration

package tls_test

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/samber/lo"
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

		body := result.BodyString()

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "secure", body)
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

	t.Run("handshake for a host outside the mappings is rejected", func(t *testing.T) {
		transport, ok := env.Client.Transport.(*http.Transport)
		require.True(t, ok)

		transport = transport.Clone()
		transport.TLSClientConfig.ServerName = "unmapped.local"

		req := integration.NewRequest(t, http.MethodGet, env.URL("tls.local", "/"))

		resp, err := (&http.Client{Transport: transport}).Do(req) //nolint:bodyclose
		require.Error(t, err)
		require.Nil(t, resp)

		var certErr x509.UnknownAuthorityError
		assert.NotErrorAs(t, err, &certErr)

		assert.EventuallyWithT(t, func(collect *assert.CollectT) {
			notices := lo.FilterMap(env.Events(), func(event events.Event, _ int) (string, bool) {
				notice, ok := event.(events.Notice)

				return notice.Text, ok
			})
			assert.Contains(collect, notices, "TLS: no https mapping for host: unmapped.local")
		}, time.Second, 10*time.Millisecond)
	})
}
