//go:build integration

package reload_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/certs"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/engine"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errUnexpectedResponse = errors.New("unexpected response")

const (
	shutdownWait = 5 * time.Second
	dialTimeout  = time.Second
)

type proxy struct {
	engine *engine.Engine
	pool   *x509.CertPool
}

func start(t *testing.T, cfg *config.UncorsConfig) *proxy {
	t.Helper()

	t.Setenv("HOME", t.TempDir())

	fs := afero.NewMemMapFs()

	caDir, err := certs.Dir()
	require.NoError(t, err)

	certPath, keyPath, err := certs.Generate(fs, caDir, 30)
	require.NoError(t, err)

	caCert, _, err := certs.Load(fs, certPath, keyPath)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	instance := engine.New(fs, engine.WithPublisher(events.NewBus()))
	require.NoError(t, instance.Start(t.Context(), cfg))

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownWait)
		defer cancel()

		assert.NoError(t, instance.Shutdown(ctx))
		instance.Wait()
	})

	return &proxy{engine: instance, pool: pool}
}

func (p *proxy) client() *http.Client {
	return &http.Client{Transport: &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: p.pool},
	}}
}

func (p *proxy) get(t *testing.T, url string) (int, string, error) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
	if err != nil {
		return 0, "", err
	}

	response, err := p.client().Do(request)
	if err != nil {
		return 0, "", err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	return response.StatusCode, string(body), err
}

func mapping(from string, port int, backend *integration.Backend) config.Mapping {
	return config.Mapping{
		From: hosts.Parse(from + "://localhost:" + strconv.Itoa(port)),
		To:   backend.AsHost(),
	}
}

func configuration(mappings ...config.Mapping) *config.UncorsConfig {
	return &config.UncorsConfig{
		Mappings: mappings,
		CacheConfig: config.CacheConfig{
			ExpirationTime: config.DefaultExpirationTime,
			MaxSize:        config.DefaultMaxSize,
			Methods:        []string{http.MethodGet},
		},
	}
}

func url(scheme string, port int, path string) string {
	return scheme + "://localhost:" + strconv.Itoa(port) + path
}

func refused(t *testing.T, port int) bool {
	t.Helper()

	dialer := &net.Dialer{Timeout: dialTimeout}

	conn, err := dialer.DialContext(t.Context(), "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return true
	}

	_ = conn.Close()

	return false
}

func TestReloadAcceptsConnections(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	port := testutils.GetFreePort(t)
	cfg := configuration(mapping("http", port, backend))
	instance := start(t, cfg)

	var (
		stop       atomic.Bool
		requests   atomic.Int64
		failures   []error
		failuresMu sync.Mutex
		wait       sync.WaitGroup
	)

	for range 4 {
		wait.Go(func() {
			for !stop.Load() {
				code, body, err := instance.get(t, url("http", port, "/"))
				if err == nil && (code != http.StatusOK || body != "ok") {
					err = fmt.Errorf("%w: %d %s", errUnexpectedResponse, code, body)
				}

				if err != nil {
					failuresMu.Lock()

					failures = append(failures, err)
					failuresMu.Unlock()
				}

				requests.Add(1)
			}
		})
	}

	for range 20 {
		require.NoError(t, instance.engine.Restart(t.Context(), cfg))
		time.Sleep(10 * time.Millisecond)
	}

	stop.Store(true)
	wait.Wait()

	assert.Positive(t, requests.Load())
	assert.Empty(t, failures, "first failures: %v", failures[:min(3, len(failures))])
}

func TestReloadCompletesInFlightRequests(t *testing.T) {
	tests := []struct {
		name string
		next func(ports []int, backend *integration.Backend) *config.UncorsConfig
	}{
		{
			name: "on a kept port",
			next: func(ports []int, backend *integration.Backend) *config.UncorsConfig {
				return configuration(mapping("http", ports[0], backend), mapping("http", ports[1], backend))
			},
		},
		{
			name: "on a removed port",
			next: func(ports []int, backend *integration.Backend) *config.UncorsConfig {
				return configuration(mapping("http", ports[1], backend))
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			entered := make(chan struct{})
			release := make(chan struct{})

			var once sync.Once

			backend := integration.NewBackend(t, func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != "/slow" {
					_, _ = io.WriteString(writer, "fast")

					return
				}

				once.Do(func() { close(entered) })
				<-release

				_, _ = io.WriteString(writer, "slow")
			})

			ports := testutils.GetFreePorts(t, 2)
			instance := start(t, configuration(mapping("http", ports[0], backend)))

			type result struct {
				code int
				body string
				err  error
			}

			slow := make(chan result, 1)

			go func() {
				code, body, err := instance.get(t, url("http", ports[0], "/slow"))
				slow <- result{code: code, body: body, err: err}
			}()

			<-entered

			require.NoError(t, instance.engine.Restart(t.Context(), testCase.next(ports, backend)))

			code, body, err := instance.get(t, url("http", ports[1], "/fast"))
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, code)
			assert.Equal(t, "fast", body)

			close(release)

			finished := <-slow
			require.NoError(t, finished.err)
			assert.Equal(t, http.StatusOK, finished.code)
			assert.Equal(t, "slow", finished.body)
		})
	}
}

func TestReloadMovesPorts(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	ports := testutils.GetFreePorts(t, 2)
	instance := start(t, configuration(mapping("http", ports[0], backend)))

	require.NoError(t, instance.engine.Restart(t.Context(), configuration(mapping("http", ports[1], backend))))

	assert.True(t, refused(t, ports[0]), "the removed port must be closed")

	code, body, err := instance.get(t, url("http", ports[1], "/"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "ok", body)
}

func TestReloadSwitchesScheme(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	port := testutils.GetFreePort(t)
	instance := start(t, configuration(mapping("http", port, backend)))

	assertServes := func(t *testing.T, scheme string) {
		t.Helper()

		code, body, err := instance.get(t, url(scheme, port, "/"))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, "ok", body)
	}

	assertServes(t, "http")

	require.NoError(t, instance.engine.Restart(t.Context(), configuration(mapping("https", port, backend))))

	assertServes(t, "https")

	code, _, err := instance.get(t, url("http", port, "/"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, code)

	require.NoError(t, instance.engine.Restart(t.Context(), configuration(mapping("http", port, backend))))

	assertServes(t, "http")

	_, _, err = instance.get(t, url("https", port, "/"))
	require.Error(t, err)
}
