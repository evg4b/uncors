package testing

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/evg4b/uncors/internal/app"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIntegration(t *testing.T) {
	go app.Uncors([]string{
		"--from", "*.*.*.*",
		"--to", "https://jsonplaceholder.typicode.com",
		"--http-port", "9023",
	}, "x.x.x")

	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("demo"))
	}))

	rootCAs, err := x509.SystemCertPool()
	testutils.CheckNoError(t, err)
	rootCAs.AddCert(tlsServer.Certificate())
	config := &tls.Config{
		RootCAs: rootCAs,
	}
	tr := &http.Transport{TLSClientConfig: config}
	client := &http.Client{Transport: tr}

	t.Run("demo", func(t *testing.T) {
		get, err := client.Get(tlsServer.URL)
		testutils.CheckNoError(t, err)

		d, err := io.ReadAll(get.Body)
		testutils.CheckNoError(t, err)

		assert.NotEmpty(t, string(d), "demo")
	})

	t.Cleanup(func() {
		tlsServer.Close()
	})
}
