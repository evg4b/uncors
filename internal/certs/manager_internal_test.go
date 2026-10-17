package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConn struct {
	localAddr net.Addr
}

func (m *mockConn) Read(_ []byte) (int, error)       { return 0, nil }
func (m *mockConn) Write(_ []byte) (int, error)      { return 0, nil }
func (m *mockConn) Close() error                     { return nil }
func (m *mockConn) LocalAddr() net.Addr              { return m.localAddr }
func (m *mockConn) RemoteAddr() net.Addr             { return nil }
func (m *mockConn) SetDeadline(time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(time.Time) error { return nil }

func TestManager_GetCertificate(t *testing.T) {
	t.Run("should auto-generate certificate when CA exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

		caDir := filepath.Join(fakeHome, ".config", "uncors")
		_, _, err := Generate(fs, caDir, 365)
		require.NoError(t, err)

		manager := NewManager(fs, events.NoopPublisher{})
		require.NotNil(t, manager)
		manager.SetHosts([]string{"localhost"})

		cert, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "localhost"})
		require.NoError(t, err)
		require.NotNil(t, cert)

		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)
		assert.Contains(t, x509Cert.DNSNames, "localhost")
	})

	t.Run("should return error when CA does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

		manager := NewManager(fs, events.NoopPublisher{})
		manager.SetHosts([]string{"test"})

		cert, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "test"})

		require.Error(t, err)
		assert.Nil(t, cert)
		assert.Contains(t, err.Error(), "failed to load CA certificate")
	})

	t.Run("should generate certificate with correct host from mapping", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

		caDir := filepath.Join(fakeHome, ".config", "uncors")
		_, _, err := Generate(fs, caDir, 365)
		require.NoError(t, err)

		testHost := "example.local"

		manager := NewManager(fs, events.NoopPublisher{})
		manager.SetHosts([]string{testHost})

		cert, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: testHost})
		require.NoError(t, err)
		require.NotNil(t, cert)

		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)
		assert.Contains(t, x509Cert.DNSNames, testHost)
	})

	t.Run("should use SNI to serve different certificates for different hosts on same port", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

		caDir := filepath.Join(fakeHome, ".config", "uncors")
		_, _, err := Generate(fs, caDir, 365)
		require.NoError(t, err)

		manager := NewManager(fs, events.NoopPublisher{})
		require.NotNil(t, manager)
		manager.SetHosts([]string{"api.local", "app.local"})

		apiCert, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "api.local"})
		require.NoError(t, err)
		require.NotNil(t, apiCert)

		apiX509Cert, err := x509.ParseCertificate(apiCert.Certificate[0])
		require.NoError(t, err)
		assert.Contains(t, apiX509Cert.DNSNames, "api.local")

		appCert, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "app.local"})
		require.NoError(t, err)
		require.NotNil(t, appCert)

		appX509Cert, err := x509.ParseCertificate(appCert.Certificate[0])
		require.NoError(t, err)
		assert.Contains(t, appX509Cert.DNSNames, "app.local")

		assert.NotContains(t, apiX509Cert.DNSNames, "app.local", "api cert should not contain app.local")
		assert.NotContains(t, appX509Cert.DNSNames, "api.local", "app cert should not contain api.local")
	})
}

func TestGetCertificate_EmptySNI(t *testing.T) {
	t.Run("extracts IP from connection when SNI is empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

		caDir := filepath.Join(fakeHome, ".config", "uncors")
		_, _, err := Generate(fs, caDir, 365)
		require.NoError(t, err)

		manager := NewManager(fs, events.NoopPublisher{})
		manager.SetHosts([]string{"192.168.1.100"})

		mockConn := &mockConn{localAddr: &net.TCPAddr{IP: net.ParseIP("192.168.1.100"), Port: 8443}}

		cert, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "", Conn: mockConn})

		require.NoError(t, err)
		require.NotNil(t, cert)

		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)
		require.Len(t, x509Cert.IPAddresses, 1)
		assert.Equal(t, "192.168.1.100", x509Cert.IPAddresses[0].String())
	})

	t.Run("returns error when SNI is empty and no connection", func(t *testing.T) {
		tmpDir := t.TempDir()
		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

		caDir := filepath.Join(fakeHome, ".config", "uncors")
		_, _, err := Generate(fs, caDir, 365)
		require.NoError(t, err)

		manager := NewManager(fs, events.NoopPublisher{})

		cert, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: ""})

		require.Error(t, err)
		assert.Nil(t, cert)
		assert.ErrorIs(t, err, ErrNoSNIProvided)
	})
}

func newManagerWithCA(t *testing.T, hosts ...string) *Manager {
	t.Helper()

	fakeHome := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", fakeHome)

	fs := afero.NewOsFs()
	require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

	caDir := filepath.Join(fakeHome, ".config", "uncors")
	_, _, err := Generate(fs, caDir, 365)
	require.NoError(t, err)

	manager := NewManager(fs, events.NoopPublisher{})
	manager.SetHosts(hosts)

	return manager
}

func TestManager_Caching(t *testing.T) {
	t.Run("should reuse the cached certificate for the same host", func(t *testing.T) {
		manager := newManagerWithCA(t, "cache.local")

		first, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "cache.local"})
		require.NoError(t, err)

		second, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "cache.local"})
		require.NoError(t, err)

		assert.Same(t, first, second)
	})

	t.Run("should generate distinct certificates for different hosts", func(t *testing.T) {
		manager := newManagerWithCA(t, "host1.local", "host2.local")

		first, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "host1.local"})
		require.NoError(t, err)

		second, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "host2.local"})
		require.NoError(t, err)

		assert.NotSame(t, first, second)
	})
}

func TestManager_Concurrent(t *testing.T) {
	t.Run("should handle concurrent requests for the same host", func(t *testing.T) {
		manager := newManagerWithCA(t, "concurrent.local")

		const numGoroutines = 10

		results := make(chan error, numGoroutines)
		for range numGoroutines {
			go func() {
				_, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "concurrent.local"})
				results <- err
			}()
		}

		for range numGoroutines {
			assert.NoError(t, <-results)
		}
	})

	t.Run("should handle concurrent requests for different hosts", func(t *testing.T) {
		manager := newManagerWithCA(t, "{name}.local")

		const numGoroutines = 5

		results := make(chan error, numGoroutines)
		for i := range numGoroutines {
			host := "host" + string(rune('0'+i)) + ".local"
			go func(h string) {
				_, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: h})
				results <- err
			}(host)
		}

		for range numGoroutines {
			assert.NoError(t, <-results)
		}
	})
}

func TestManager_Notices(t *testing.T) {
	t.Run("reports a missing CA once", func(t *testing.T) {
		t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))

		publisher := &testutils.EventRecorder{}
		manager := NewManager(afero.NewMemMapFs(), publisher)
		manager.SetHosts([]string{"localhost"})

		for range 3 {
			_, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "localhost"})
			require.Error(t, err)
		}

		notices := publisher.Notices()
		require.Len(t, notices, 1)
		assert.Equal(t, events.NoticeError, notices[0].Level)
		assert.Contains(t, notices[0].Text, "TLS: failed to load CA certificate")
	})

	t.Run("warns once about a CA that expires soon and keeps signing", func(t *testing.T) {
		fakeHome := filepath.Join(t.TempDir(), "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		_, _, err := Generate(fs, filepath.Join(fakeHome, ".config", "uncors"), 3)
		require.NoError(t, err)

		publisher := &testutils.EventRecorder{}
		manager := NewManager(fs, publisher)
		manager.SetHosts([]string{"one.local", "two.local"})

		for _, host := range []string{"one.local", "two.local"} {
			cert, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: host})
			require.NoError(t, err)
			assert.NotNil(t, cert)
		}

		notices := publisher.Notices()
		require.Len(t, notices, 1)
		assert.Equal(t, events.NoticeWarning, notices[0].Level)
		assert.Contains(t, notices[0].Text, "CA certificate expires in")
	})

	t.Run("reports a certificate failure once per host", func(t *testing.T) {
		manager := newManagerWithCA(t, "one.local", "two.local")
		publisher := &testutils.EventRecorder{}
		manager.publisher = publisher

		require.NoError(t, manager.ensureCA(t.Context()))

		// A key that does not match the CA makes every signature fail.
		otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		manager.caKey = otherKey

		for _, host := range []string{"one.local", "one.local", "two.local", "one.local"} {
			_, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: host})
			require.Error(t, err)
		}

		notices := publisher.Notices()
		require.Len(t, notices, 2)
		assert.Contains(t, notices[0].Text, "one.local")
		assert.Contains(t, notices[1].Text, "two.local")
	})
}

func TestManager_Hosts(t *testing.T) {
	t.Run("rejects a host outside the mappings and reports it once", func(t *testing.T) {
		manager := newManagerWithCA(t, "app.local")
		publisher := &testutils.EventRecorder{}
		manager.publisher = publisher

		for _, host := range []string{"evil.local", "evil.local", "other.local"} {
			cert, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: host})
			require.ErrorIs(t, err, ErrHostNotMapped)
			assert.Nil(t, cert)
		}

		assert.Empty(t, manager.cache)

		notices := publisher.Notices()
		require.Len(t, notices, 2)
		assert.Equal(t, events.NoticeError, notices[0].Level)
		assert.Equal(t, "TLS: no https mapping for host: evil.local", notices[0].Text)
		assert.Equal(t, "TLS: no https mapping for host: other.local", notices[1].Text)
	})

	t.Run("rejects every host before hosts are set", func(t *testing.T) {
		manager := newManagerWithCA(t)

		_, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "localhost"})
		require.ErrorIs(t, err, ErrHostNotMapped)
	})

	t.Run("rejects a host without loading the CA", func(t *testing.T) {
		t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))

		publisher := &testutils.EventRecorder{}
		manager := NewManager(afero.NewMemMapFs(), publisher)
		manager.SetHosts([]string{"app.local"})

		_, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "evil.local"})
		require.ErrorIs(t, err, ErrHostNotMapped)

		require.Len(t, publisher.Notices(), 1)
		assert.NotContains(t, publisher.Notices()[0].Text, "CA")
	})

	t.Run("matches a placeholder against one label", func(t *testing.T) {
		manager := newManagerWithCA(t, "{tenant}.local.dev")

		cert, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "acme.local.dev"})
		require.NoError(t, err)

		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)
		assert.Equal(t, []string{"acme.local.dev"}, x509Cert.DNSNames)

		_, err = manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "a.b.local.dev"})
		require.ErrorIs(t, err, ErrHostNotMapped)
	})

	t.Run("uses the local address without SNI only when it is mapped", func(t *testing.T) {
		manager := newManagerWithCA(t, "localhost")
		conn := &mockConn{localAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8443}}

		_, err := manager.GetCertificate(&tls.ClientHelloInfo{Conn: conn})
		require.ErrorIs(t, err, ErrHostNotMapped)

		manager.SetHosts([]string{"127.0.0.1"})

		cert, err := manager.GetCertificate(&tls.ClientHelloInfo{Conn: conn})
		require.NoError(t, err)

		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)
		require.Len(t, x509Cert.IPAddresses, 1)
		assert.Equal(t, "127.0.0.1", x509Cert.IPAddresses[0].String())
	})

	t.Run("replaces the hosts and drops certificates of removed ones", func(t *testing.T) {
		manager := newManagerWithCA(t, "one.local")
		publisher := &testutils.EventRecorder{}
		manager.publisher = publisher

		_, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "one.local"})
		require.NoError(t, err)

		_, err = manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "two.local"})
		require.ErrorIs(t, err, ErrHostNotMapped)

		manager.SetHosts([]string{"two.local"})

		assert.NotContains(t, manager.cache, "one.local")

		_, err = manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "one.local"})
		require.ErrorIs(t, err, ErrHostNotMapped)

		_, err = manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "two.local"})
		require.NoError(t, err)

		assert.Len(t, publisher.Notices(), 2)
	})

	t.Run("keeps a bounded number of certificates", func(t *testing.T) {
		manager := newManagerWithCA(t, "{name}.local")

		for i := range maxCachedCertificates + 10 {
			_, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: fmt.Sprintf("host%d.local", i)})
			require.NoError(t, err)
		}

		assert.Len(t, manager.cache, maxCachedCertificates)
	})

	t.Run("stops reporting after too many rejected hosts", func(t *testing.T) {
		manager := newManagerWithCA(t, "app.local")
		publisher := &testutils.EventRecorder{}
		manager.publisher = publisher

		for i := range maxReportedHosts + 10 {
			_, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: fmt.Sprintf("host%d.evil", i)})
			require.ErrorIs(t, err, ErrHostNotMapped)
		}

		assert.Len(t, publisher.Notices(), maxReportedHosts)
	})
}

func TestManager_SingleIssuance(t *testing.T) {
	recorded := testutils.Telemetry(t)
	manager := newManagerWithCA(t, "burst.local")

	const handshakes = 50

	var (
		start sync.WaitGroup
		done  sync.WaitGroup
	)

	start.Add(1)

	certificates := make(chan *tls.Certificate, handshakes)
	for range handshakes {
		done.Go(func() {
			start.Wait()

			cert, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "burst.local"})
			assert.NoError(t, err)

			certificates <- cert
		})
	}

	start.Done()
	done.Wait()
	close(certificates)

	unique := make(map[*tls.Certificate]struct{}, handshakes)
	for cert := range certificates {
		unique[cert] = struct{}{}
	}

	assert.Len(t, unique, 1)

	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.tls.certificates"))
}
