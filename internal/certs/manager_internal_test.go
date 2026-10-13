package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"net"
	"path/filepath"
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

func newManagerWithCA(t *testing.T) *Manager {
	t.Helper()

	fakeHome := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", fakeHome)

	fs := afero.NewOsFs()
	require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

	caDir := filepath.Join(fakeHome, ".config", "uncors")
	_, _, err := Generate(fs, caDir, 365)
	require.NoError(t, err)

	return NewManager(fs, events.NoopPublisher{})
}

func TestManager_Caching(t *testing.T) {
	t.Run("should reuse the cached certificate for the same host", func(t *testing.T) {
		manager := newManagerWithCA(t)

		first, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "cache.local"})
		require.NoError(t, err)

		second, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "cache.local"})
		require.NoError(t, err)

		assert.Same(t, first, second)
	})

	t.Run("should generate distinct certificates for different hosts", func(t *testing.T) {
		manager := newManagerWithCA(t)

		first, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "host1.local"})
		require.NoError(t, err)

		second, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "host2.local"})
		require.NoError(t, err)

		assert.NotSame(t, first, second)
	})
}

func TestManager_Concurrent(t *testing.T) {
	t.Run("should handle concurrent requests for the same host", func(t *testing.T) {
		manager := newManagerWithCA(t)

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
		manager := newManagerWithCA(t)

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
		manager := newManagerWithCA(t)
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
