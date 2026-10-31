//go:build integration

package integration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/certs"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/engine"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// caValidityDays must stay clear of the proxy's expiration warning threshold
// (7 days); below it every run publishes a warning notice.
const (
	caValidityDays    = 30
	configFilePerm    = 0o600
	configPath        = "/uncors-config.yaml"
	proxyShutdownWait = 5 * time.Second
)

type Env struct {
	Client  *http.Client
	Backend *Backend
	FS      afero.Fs

	routes []route
	events *eventRecorder
	engine *engine.Engine
}

type route struct {
	match  func(string) bool
	port   int
	scheme string
}

type Option func(afero.Fs) error

func WithFile(path, content string) Option {
	return func(fs afero.Fs) error {
		return afero.WriteFile(fs, path, []byte(content), os.ModePerm)
	}
}

// New starts the proxy for cfg on an in-memory filesystem. A From without a
// port gets a fresh one; an explicit port lets several mappings share a listener.
func New(t *testing.T, backend *Backend, cfg *config.UncorsConfig, opts ...Option) *Env {
	t.Helper()

	env := &Env{Backend: backend, FS: afero.NewMemMapFs()}

	for _, opt := range opts {
		require.NoError(t, opt(env.FS))
	}

	for i := range cfg.Mappings {
		from := &cfg.Mappings[i].From
		require.NotEmpty(t, from.Hostname, "mapping From host must be a named host, not empty")

		if from.Port == "" {
			from.Port = strconv.Itoa(testutils.GetFreePort(t))
		}

		port, err := strconv.Atoi(from.Port)
		require.NoError(t, err)

		env.routes = append(env.routes, route{match: matchHost(from.Hostname), port: port, scheme: from.Scheme})
	}

	caDir, err := certs.Dir()
	require.NoError(t, err)

	certPath, keyPath, err := certs.Generate(env.FS, caDir, caValidityDays)
	require.NoError(t, err)

	caCert, _, err := certs.Load(env.FS, certPath, keyPath)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	env.Client = &http.Client{
		Transport: &http.Transport{
			DialContext:     Dial,
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool},
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	bus := events.NewBus()
	env.events = recordEvents(bus.Subscribe(eventsBuffer))
	env.engine = engine.New(env.FS, engine.WithPublisher(bus))

	require.NoError(t, env.engine.Start(t.Context(), env.load(t, cfg)))

	// Shutting down closes the generation, which flushes HAR files; cleanups
	// run in reverse order, so this happens before temp dirs are removed.
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), proxyShutdownWait)
		defer cancel()

		assert.NoError(t, env.engine.Shutdown(ctx))
		env.engine.Wait()
		bus.Close()
		<-env.events.done
	})

	return env
}

func Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(config.ListenAddress, port))
}

func (e *Env) Restart(t *testing.T, cfg *config.UncorsConfig) {
	t.Helper()

	require.NoError(t, e.TryRestart(t, cfg))
}

func (e *Env) TryRestart(t *testing.T, cfg *config.UncorsConfig) error {
	t.Helper()

	return e.engine.Restart(t.Context(), e.load(t, cfg))
}

func (e *Env) Shutdown(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), proxyShutdownWait)
	defer cancel()

	require.NoError(t, e.engine.Shutdown(ctx))
}

func (e *Env) URL(host, path string) string {
	matched := e.routeFor(host)
	authority := net.JoinHostPort(host, strconv.Itoa(matched.port))

	return testutils.JoinPath(matched.scheme+"://"+authority, path)
}

func (e *Env) RawURL(host, pathAndQuery string) string {
	return strings.TrimSuffix(e.URL(host, "/"), "/") + pathAndQuery
}

func (e *Env) PortFor(host string) int {
	return e.routeFor(host).port
}

func (e *Env) routeFor(host string) route {
	for _, r := range e.routes {
		if r.match(host) {
			return r
		}
	}

	panic("integration: no mapping registered for host " + host)
}

func (e *Env) load(t *testing.T, cfg *config.UncorsConfig) *config.UncorsConfig {
	t.Helper()

	require.NoError(t, afero.WriteFile(e.FS, configPath, marshalConfig(t, cfg), configFilePerm))

	loaded, _, err := config.LoadConfiguration(t.Context(), e.FS, nil, []string{"-c", configPath})
	require.NoError(t, err)

	return loaded
}

// Scripts are trimmed: yaml.v3 emits a leading newline as a "|4" block scalar
// that does not parse back.
func marshalConfig(t *testing.T, cfg *config.UncorsConfig) []byte {
	t.Helper()

	normalised := *cfg
	normalised.Mappings = make(config.Mappings, 0, len(cfg.Mappings))

	for _, mapping := range cfg.Mappings {
		mapping = mapping.Clone()
		for i := range mapping.Scripts {
			mapping.Scripts[i].Script = strings.TrimSpace(mapping.Scripts[i].Script)
		}

		normalised.Mappings = append(normalised.Mappings, mapping)
	}

	data, err := yaml.Marshal(&normalised)
	require.NoError(t, err)

	return data
}

func matchHost(pattern string) func(string) bool {
	labels := strings.Split(pattern, ".")
	for i, label := range labels {
		if strings.ContainsAny(label, "{*") {
			labels[i] = "[^.]+"
		} else {
			labels[i] = regexp.QuoteMeta(label)
		}
	}

	return regexp.MustCompile("(?i)^" + strings.Join(labels, `\.`) + "$").MatchString
}
