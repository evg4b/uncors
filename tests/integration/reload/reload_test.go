//go:build integration

package reload_test

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errUnexpectedResponse = errors.New("unexpected response")

const dialTimeout = time.Second

type proxy struct {
	*integration.Env

	client *http.Client
}

func start(t *testing.T, backend *integration.Backend, cfg *config.UncorsConfig) *proxy {
	t.Helper()

	env := integration.New(t, backend, cfg)
	transport, ok := env.Client.Transport.(*http.Transport)
	require.True(t, ok)

	transport = transport.Clone()
	transport.DisableKeepAlives = true

	return &proxy{Env: env, client: &http.Client{Transport: transport}}
}

func (p *proxy) get(t *testing.T, url string) (int, string, error) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
	if err != nil {
		return 0, "", err
	}

	response, err := p.client.Do(request)
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
	return &config.UncorsConfig{Mappings: mappings}
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
	instance := start(t, backend, cfg)

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
		instance.Restart(t, cfg)
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
			instance := start(t, backend, configuration(mapping("http", ports[0], backend)))

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

			instance.Restart(t, testCase.next(ports, backend))

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

func TestReloadKeepsNewHARRecording(t *testing.T) {
	const harFile = "/session.har"

	entered := make(chan struct{})
	release := make(chan struct{})

	backend := integration.NewBackend(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/slow" {
			close(entered)
			<-release
		}

		_, _ = io.WriteString(writer, "ok")
	})

	port := testutils.GetFreePort(t)
	harMapping := mapping("http", port, backend)
	harMapping.HAR = config.HARConfig{File: harFile}
	cfg := configuration(harMapping)
	instance := start(t, backend, cfg)

	slow := make(chan error, 1)

	go func() {
		_, _, err := instance.get(t, url("http", port, "/slow"))
		slow <- err
	}()

	<-entered

	instance.Restart(t, cfg)

	_, _, err := instance.get(t, url("http", port, "/fast"))
	require.NoError(t, err)

	recorded := func(path string) bool {
		data, err := afero.ReadFile(instance.FS, harFile)

		return err == nil && strings.Contains(string(data), path)
	}

	require.Eventually(t, func() bool { return recorded("/fast") }, 3*time.Second, 50*time.Millisecond)

	close(release)
	require.NoError(t, <-slow)

	assert.Never(t, func() bool { return !recorded("/fast") }, 2*time.Second, 50*time.Millisecond)
}

func TestReloadMovesPorts(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	ports := testutils.GetFreePorts(t, 2)
	instance := start(t, backend, configuration(mapping("http", ports[0], backend)))

	instance.Restart(t, configuration(mapping("http", ports[1], backend)))

	assert.True(t, refused(t, ports[0]), "the removed port must be closed")

	code, body, err := instance.get(t, url("http", ports[1], "/"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "ok", body)
}

func TestReloadSwitchesScheme(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	port := testutils.GetFreePort(t)
	instance := start(t, backend, configuration(mapping("http", port, backend)))

	assertServes := func(t *testing.T, scheme string) {
		t.Helper()

		code, body, err := instance.get(t, url(scheme, port, "/"))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, "ok", body)
	}

	assertServes(t, "http")

	instance.Restart(t, configuration(mapping("https", port, backend)))

	assertServes(t, "https")

	code, _, err := instance.get(t, url("http", port, "/"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, code)

	instance.Restart(t, configuration(mapping("http", port, backend)))

	assertServes(t, "http")

	_, _, err = instance.get(t, url("https", port, "/"))
	require.Error(t, err)
}
