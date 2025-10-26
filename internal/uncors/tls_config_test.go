package uncors

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTLSConfig(t *testing.T) {
	t.Run("should return error when no mappings provided", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mappings := config.Mappings{}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		assert.Error(t, err)
		assert.Nil(t, tlsConfig)
		assert.Equal(t, infratls.ErrNoMappingsProvided, err)
	})

	t.Run("should load custom certificate when cert and key files provided", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		certPath := "/test/cert.crt"
		keyPath := "/test/key.key"

		tmpDir, err := os.MkdirTemp("", "tls-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		caConfig := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    tmpDir,
		}
		generatedCertPath, generatedKeyPath, err := infratls.GenerateCA(caConfig)
		require.NoError(t, err)

		certData, err := os.ReadFile(generatedCertPath)
		require.NoError(t, err)
		keyData, err := os.ReadFile(generatedKeyPath)
		require.NoError(t, err)

		require.NoError(t, afero.WriteFile(fs, certPath, certData, 0o644))
		require.NoError(t, afero.WriteFile(fs, keyPath, keyData, 0o600))

		mappings := config.Mappings{
			{
				From:     "https://localhost:8443",
				To:       "http://example.com",
				CertFile: certPath,
				KeyFile:  keyPath,
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.NoError(t, err)
		require.NotNil(t, tlsConfig)
		assert.NotEmpty(t, tlsConfig.Certificates)
		assert.Equal(t, uint16(0x0303), tlsConfig.MinVersion) // TLS 1.2
	})

	t.Run("should return error when cert file does not exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		mappings := config.Mappings{
			{
				From:     "https://localhost:8443",
				To:       "http://example.com",
				CertFile: "/nonexistent/cert.crt",
				KeyFile:  "/nonexistent/key.key",
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		assert.Error(t, err)
		assert.Nil(t, tlsConfig)
		assert.Contains(t, err.Error(), "failed to read certificate file")
	})

	t.Run("should return error when key file does not exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		certPath := "/test/cert.crt"
		require.NoError(t, afero.WriteFile(fs, certPath, []byte("cert data"), 0o644))

		mappings := config.Mappings{
			{
				From:     "https://localhost:8443",
				To:       "http://example.com",
				CertFile: certPath,
				KeyFile:  "/nonexistent/key.key",
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		assert.Error(t, err)
		assert.Nil(t, tlsConfig)
		assert.Contains(t, err.Error(), "failed to read key file")
	})

	t.Run("should return error for invalid certificate data", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		certPath := "/test/cert.crt"
		keyPath := "/test/key.key"

		require.NoError(t, afero.WriteFile(fs, certPath, []byte("invalid cert"), 0o644))
		require.NoError(t, afero.WriteFile(fs, keyPath, []byte("invalid key"), 0o600))

		mappings := config.Mappings{
			{
				From:     "https://localhost:8443",
				To:       "http://example.com",
				CertFile: certPath,
				KeyFile:  keyPath,
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		assert.Error(t, err)
		assert.Nil(t, tlsConfig)
		assert.Contains(t, err.Error(), "failed to load certificate")
	})

	t.Run("should auto-generate certificate when no custom cert provided and CA exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

		caDir := filepath.Join(fakeHome, ".config", "uncors")
		caConfig := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
			Fs:           fs,
		}
		_, _, err := infratls.GenerateCA(caConfig)
		require.NoError(t, err)

		mappings := config.Mappings{
			{
				From: "https://localhost:8443",
				To:   "http://example.com",
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.NoError(t, err)
		require.NotNil(t, tlsConfig)
		assert.NotEmpty(t, tlsConfig.Certificates)
		assert.Equal(t, uint16(0x0303), tlsConfig.MinVersion) // TLS 1.2

		cert := tlsConfig.Certificates[0]
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)
		assert.Contains(t, x509Cert.DNSNames, "localhost")
	})

	t.Run("should return error when CA does not exist and no custom cert provided", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

		mappings := config.Mappings{
			{
				From: "https://localhost:8443",
				To:   "http://example.com",
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		assert.Error(t, err)
		assert.Nil(t, tlsConfig)
		assert.Contains(t, err.Error(), "failed to load CA certificate")
	})

	t.Run("should return error when mapping has invalid host", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

		caDir := filepath.Join(fakeHome, ".config", "uncors")
		caConfig := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
			Fs:           fs,
		}
		_, _, err := infratls.GenerateCA(caConfig)
		require.NoError(t, err)

		mappings := config.Mappings{
			{
				From: "https://[invalid-host",
				To:   "http://example.com",
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		assert.Error(t, err)
		assert.Nil(t, tlsConfig)
		assert.Contains(t, err.Error(), "failed to parse mapping host")
	})

	t.Run("should generate certificate with correct host from mapping", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

		caDir := filepath.Join(fakeHome, ".config", "uncors")
		caConfig := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
			Fs:           fs,
		}
		_, _, err := infratls.GenerateCA(caConfig)
		require.NoError(t, err)

		testHost := "example.local"
		mappings := config.Mappings{
			{
				From: "https://" + testHost + ":8443",
				To:   "http://example.com",
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.NoError(t, err)
		require.NotNil(t, tlsConfig)

		cert := tlsConfig.Certificates[0]
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)
		assert.Contains(t, x509Cert.DNSNames, testHost)
	})
}
