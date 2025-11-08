package server_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	infraTls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/phayes/freeport"
	"github.com/samber/lo"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createServerCert(t *testing.T, caCert *x509.Certificate, caKey *rsa.PrivateKey, host string) tls.Certificate {
	t.Helper()

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); ip != nil {
		tmpl.IPAddresses = []net.IP{ip}
	} else {
		tmpl.DNSNames = []string{host}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, caCert, &serverKey.PublicKey, caKey)
	require.NoError(t, err)

	// кодируем в PEM для tls.X509KeyPair
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	return cert
}

func TestServer(t *testing.T) {
	const porstCount = 5

	expectedContent := "Test"
	handler := contracts.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, expectedContent)
		require.NoError(t, err)
	})

	assertResponse := func(t *testing.T, url string, pool *x509.CertPool) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, err)

		var client *http.Client

		if pool != nil {
			client = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						MinVersion: tls.VersionTLS13,
						RootCAs:    pool,
						ServerName: "127.0.0.1",
					},
				},
			}
		} else {
			client = http.DefaultClient
		}

		response, err := client.Do(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)

		bodyData, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, expectedContent, string(bodyData))
	}

	t.Run("multiple http ports", func(t *testing.T) {
		freePorts, err := freeport.GetFreePorts(porstCount)
		require.NoError(t, err)

		targets := lo.Map(freePorts, func(port int, _ int) server.Target {
			return server.Target{
				Address: hosts.Loopback.Port(port),
				Handler: handler,
			}
		})

		instance := server.New()
		instance.Start(t.Context(), targets)

		defer func() {
			require.NoError(t, instance.Close())
		}()

		for _, port := range freePorts {
			t.Run(fmt.Sprintf("port %d", port), func(t *testing.T) {
				assertResponse(t, hosts.Loopback.HTTPPort(port), nil)
			})
		}
	})

	t.Run("multiple https ports", func(t *testing.T) {
		freePorts, err := freeport.GetFreePorts(porstCount)
		require.NoError(t, err)

		fs := afero.NewMemMapFs()
		certPath, keyPath, err := infraTls.GenerateCA(infraTls.CAConfig{
			ValidityDays: 10,
			Fs:           fs,
			OutputDir:    ".",
		})
		require.NoError(t, err)

		caCert, caKey, err := infraTls.LoadCA(fs, certPath, keyPath)
		require.NoError(t, err)

		pool := x509.NewCertPool()
		pool.AddCert(caCert)

		targets := lo.Map(freePorts, func(port int, _ int) server.Target {
			return server.Target{
				Address: hosts.Loopback.Port(port),
				Handler: handler,
				TLSConfgi: &tls.Config{
					MinVersion: tls.VersionTLS13,
					Certificates: []tls.Certificate{
						createServerCert(t, caCert, caKey, hosts.Loopback.Host()),
					},
				},
			}
		})

		instance := server.New()
		instance.Start(t.Context(), targets)

		defer func() {
			require.NoError(t, instance.Close())
		}()

		for _, port := range freePorts {
			t.Run(fmt.Sprintf("port %d", port), func(t *testing.T) {
				assertResponse(t, hosts.Loopback.HTTPSPort(port), pool)
			})
		}
	})

	t.Run("mix of http and https ports", func(t *testing.T) {
		freeHTTPPorts, err := freeport.GetFreePorts(porstCount)
		require.NoError(t, err)

		freeHTTPSPorts, err := freeport.GetFreePorts(porstCount)
		require.NoError(t, err)

		fs := afero.NewMemMapFs()
		certPath, keyPath, err := infraTls.GenerateCA(infraTls.CAConfig{
			ValidityDays: 10,
			Fs:           fs,
			OutputDir:    ".",
		})
		require.NoError(t, err)

		caCert, caKey, err := infraTls.LoadCA(fs, certPath, keyPath)
		require.NoError(t, err)

		pool := x509.NewCertPool()
		pool.AddCert(caCert)

		httpTargets := lo.Map(freeHTTPPorts, func(port int, _ int) server.Target {
			return server.Target{
				Address: hosts.Loopback.Port(port),
				Handler: handler,
			}
		})

		httpsTargets := lo.Map(freeHTTPSPorts, func(port int, _ int) server.Target {
			return server.Target{
				Address: hosts.Loopback.Port(port),
				Handler: handler,
				TLSConfgi: &tls.Config{
					MinVersion: tls.VersionTLS13,
					Certificates: []tls.Certificate{
						createServerCert(t, caCert, caKey, hosts.Loopback.Host()),
					},
				},
			}
		})

		instance := server.New()
		instance.Start(t.Context(), append(httpTargets, httpsTargets...))

		defer func() {
			require.NoError(t, instance.Close())
		}()

		for _, port := range freeHTTPSPorts {
			t.Run(fmt.Sprintf("https port %d", port), func(t *testing.T) {
				assertResponse(t, hosts.Loopback.HTTPSPort(port), pool)
			})
		}

		for _, port := range freeHTTPPorts {
			t.Run(fmt.Sprintf("http port %d", port), func(t *testing.T) {
				assertResponse(t, hosts.Loopback.HTTPPort(port), nil)
			})
		}
	})

	t.Run("shutdown", func(t *testing.T) {
		port := freeport.GetPort()

		instance := server.New()
		instance.Start(t.Context(), []server.Target{
			{
				Address: hosts.Loopback.Port(port),
				Handler: handler,
			},
		})

		defer func() {
			require.NoError(t, instance.Close())
		}()

		assertResponse(t, hosts.Loopback.HTTPPort(port), nil)

		err := instance.Shutdown(t.Context())
		require.NoError(t, err)

		assert.True(t, IsPortFree(port))
	})

	t.Run("close", func(t *testing.T) {
		port := freeport.GetPort()

		instance := server.New()
		instance.Start(t.Context(), []server.Target{
			{
				Address: hosts.Loopback.Port(port),
				Handler: handler,
			},
		})

		defer func() {
			require.NoError(t, instance.Close())
		}()

		assertResponse(t, hosts.Loopback.HTTPPort(port), nil)

		err := instance.Close()
		require.NoError(t, err)

		assert.True(t, IsPortFree(port))
	})

	t.Run("Restart", func(t *testing.T) {
		initial := freeport.GetPort()
		restarted := freeport.GetPort()
		instance := server.New()

		instance.Start(t.Context(), []server.Target{
			{
				Address: hosts.Loopback.Port(initial),
				Handler: handler,
			},
		})

		assertResponse(t, hosts.Loopback.HTTPPort(initial), nil)
		require.True(t, IsPortFree(restarted))

		err := instance.Restart(t.Context(), []server.Target{
			{
				Address: hosts.Loopback.Port(restarted),
				Handler: handler,
			},
		})
		require.NoError(t, err)

		assert.True(t, IsPortFree(initial))
		assertResponse(t, hosts.Loopback.HTTPPort(restarted), nil)
	})

	t.Run("wait", func(t *testing.T) {
		eventsCh := make(chan string, 10)

		port := freeport.GetPort()

		instance := server.New()

		eventsCh <- "server started"

		instance.Start(t.Context(), []server.Target{
			{
				Address: hosts.Loopback.Port(port),
				Handler: contracts.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) {
					eventsCh <- "handler trigered"

					w.WriteHeader(http.StatusOK)
					_, err := fmt.Fprint(w, expectedContent)
					require.NoError(t, err)
				}),
			},
		})

		defer func() {
			require.NoError(t, instance.Close())
		}()

		go func() {
			eventsCh <- "waiting started"

			instance.Wait()

			eventsCh <- "waiting finished"
		}()

		assertResponse(t, hosts.Loopback.HTTPPort(port), nil)

		go func(t *testing.T) {
			eventsCh <- "shutdown trigered"

			err := instance.Shutdown(t.Context())
			assert.NoError(t, err)
			close(eventsCh)
		}(t)

		var events []string
		for v := range eventsCh {
			events = append(events, v)
		}

		assert.Equal(t, []string{
			"server started",
			"waiting started",
			"handler trigered",
			"shutdown trigered",
			"waiting finished",
		}, events)
	})
}

func IsPortFree(port int) bool {
	l, err := net.Listen("tcp", hosts.Loopback.Port(port)) // nolint: noctx
	if err != nil {
		return false
	}
	defer l.Close()

	return true
}
