package uncors

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"path/filepath"
	"testing"
	"time"

	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConn is a minimal net.Conn implementation for testing.
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

func TestBuildTLSConfig(t *testing.T) {
	t.Run("should auto-generate certificate when CA exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

		caDir := filepath.Join(fakeHome, ".config", "uncors")
		caConfig := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
			Fs:           fs,
		}
		_, _, err := infratls.GenerateCA(caConfig)
		require.NoError(t, err)

		tlsConfig, err := buildTLSConfig(fs)

		require.NoError(t, err)
		require.NotNil(t, tlsConfig)
		assert.NotNil(t, tlsConfig.GetCertificate)
		assert.Equal(t, uint16(0x0303), tlsConfig.MinVersion) // TLS 1.2

		// Test SNI by requesting certificate for localhost
		cert, err := tlsConfig.GetCertificate(&tls.ClientHelloInfo{ServerName: "localhost"})
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

		tlsConfig, err := buildTLSConfig(fs)

		require.Error(t, err)
		assert.Nil(t, tlsConfig)
		assert.Contains(t, err.Error(), "failed to load CA certificate")
	})

	t.Run("should generate certificate with correct host from mapping", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

		caDir := filepath.Join(fakeHome, ".config", "uncors")
		caConfig := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
			Fs:           fs,
		}
		_, _, err := infratls.GenerateCA(caConfig)
		require.NoError(t, err)

		testHost := "example.local"

		tlsConfig, err := buildTLSConfig(fs)

		require.NoError(t, err)
		require.NotNil(t, tlsConfig)

		// Test SNI by requesting certificate for the specific host
		cert, err := tlsConfig.GetCertificate(&tls.ClientHelloInfo{ServerName: testHost})
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

		// Generate CA for auto-generated certificates
		caDir := filepath.Join(fakeHome, ".config", "uncors")
		caConfig := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
			Fs:           fs,
		}
		_, _, err := infratls.GenerateCA(caConfig)
		require.NoError(t, err)

		tlsConfig, err := buildTLSConfig(fs)

		require.NoError(t, err)
		require.NotNil(t, tlsConfig)
		assert.NotNil(t, tlsConfig.GetCertificate)

		// Test SNI for api.local - should auto-generate cert for this host
		apiCert, err := tlsConfig.GetCertificate(&tls.ClientHelloInfo{ServerName: "api.local"})
		require.NoError(t, err)
		require.NotNil(t, apiCert)

		apiX509Cert, err := x509.ParseCertificate(apiCert.Certificate[0])
		require.NoError(t, err)
		assert.Contains(t, apiX509Cert.DNSNames, "api.local")

		// Test SNI for app.local - should auto-generate cert for this host
		appCert, err := tlsConfig.GetCertificate(&tls.ClientHelloInfo{ServerName: "app.local"})
		require.NoError(t, err)
		require.NotNil(t, appCert)

		appX509Cert, err := x509.ParseCertificate(appCert.Certificate[0])
		require.NoError(t, err)
		assert.Contains(t, appX509Cert.DNSNames, "app.local")

		// Verify that each certificate is valid for its respective host
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
		_, _, err := infratls.GenerateCA(infratls.CAConfig{ValidityDays: 365, OutputDir: caDir, Fs: fs})
		require.NoError(t, err)

		manager, err := newHostCertManager(fs)
		require.NoError(t, err)

		// Mock connection with IP address
		mockConn := &mockConn{localAddr: &net.TCPAddr{IP: net.ParseIP("192.168.1.100"), Port: 8443}}

		cert, err := manager.getCertificate(&tls.ClientHelloInfo{ServerName: "", Conn: mockConn})

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
		_, _, err := infratls.GenerateCA(infratls.CAConfig{ValidityDays: 365, OutputDir: caDir, Fs: fs})
		require.NoError(t, err)

		manager, err := newHostCertManager(fs)
		require.NoError(t, err)

		cert, err := manager.getCertificate(&tls.ClientHelloInfo{ServerName: ""})

		require.Error(t, err)
		assert.Nil(t, cert)
		assert.ErrorIs(t, err, infratls.ErrNoSNIProvided)
	})
}
