package uncors_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
	infraTls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/phayes/freeport"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUncorsApp(t *testing.T) {
	fs := afero.NewMemMapFs()
	app := uncors.CreateUncors(fs, log.Default(), "x.x.x")

	testResponceHeader := "# Test resrver"
	hostFmt := func(host string) string {
		return fmt.Sprintf("\tHost: %v", host)
	}
	methodFmt := func(method string) string {
		return fmt.Sprintf("\tMethod: %v", method)
	}
	urlFmt := func(method string) string {
		return fmt.Sprintf("\tURL: %v", method)
	}

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, testResponceHeader)
		fmt.Fprintln(w, methodFmt(r.Method))
		fmt.Fprintln(w, urlFmt(r.URL.String()))
		fmt.Fprintln(w, hostFmt(r.Host))
	}))

	defer func() {
		targetServer.Close()
	}()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	certPath, keyPath, err := infraTls.GenerateCA(infraTls.CAConfig{
		Fs:           fs,
		ValidityDays: 10,
		OutputDir:    filepath.Join(homeDir, ".config", "uncors"),
	})
	require.NoError(t, err)

	caCert, _, err := infraTls.LoadCA(fs, certPath, keyPath)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
				RootCAs:    pool,
				ServerName: "127.0.0.1",
			},
		},
	}

	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	err = app.Start(t.Context(), &config.UncorsConfig{
		Mappings: []config.Mapping{
			{
				From: hosts.Loopback.HTTPPort(port),
				To:   targetServer.URL,
			},
		},
	})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, app.Close())
	}()

	t.Run("proxy", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			hosts.Loopback.HTTPPort(port),
			nil,
		)
		require.NoError(t, err)

		response, err := client.Do(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)

		bodyData, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		uri, err := url.Parse(targetServer.URL)
		require.NoError(t, err)

		assert.Contains(t, string(bodyData), uri.Host)
		assert.Contains(t, string(bodyData), methodFmt(http.MethodGet))
	})
}
