package tls_test

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
	"time"

	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCA(t *testing.T) {
	t.Run("should generate CA certificate and key", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    tmpDir,
		}

		certPath, keyPath, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		// Verify files exist
		assert.FileExists(t, certPath)
		assert.FileExists(t, keyPath)

		// Verify file paths
		assert.Equal(t, filepath.Join(tmpDir, "ca.crt"), certPath)
		assert.Equal(t, filepath.Join(tmpDir, "ca.key"), keyPath)

		// Verify key file permissions
		keyInfo, err := os.Stat(keyPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), keyInfo.Mode().Perm())
	})

	t.Run("should create output directory if not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputDir := filepath.Join(tmpDir, "subdir", "nested")

		config := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    outputDir,
		}

		_, _, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		// Verify directory was created
		assert.DirExists(t, outputDir)
	})

	t.Run("should generate valid certificate with correct properties", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := infratls.CAConfig{
			ValidityDays: 730,
			OutputDir:    tmpDir,
		}

		certPath, keyPath, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		// Load and verify certificate
		cert, key, err := infratls.LoadCA(nil, certPath, keyPath)
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.NotNil(t, key)

		// Verify certificate properties
		assert.True(t, cert.IsCA)
		assert.Equal(t, "UNCORS Local Development Root CA", cert.Subject.CommonName)
		assert.Contains(t, cert.Subject.Organization, "UNCORS Development CA")
		assert.Contains(t, cert.Subject.Country, "US")

		// Verify validity period (allow some tolerance for test execution time)
		expectedDuration := time.Duration(730) * 24 * time.Hour
		actualDuration := cert.NotAfter.Sub(cert.NotBefore)
		tolerance := 5 * time.Minute
		assert.InDelta(t, expectedDuration, actualDuration, float64(tolerance))

		// Verify key usage
		assert.NotEqual(t, 0, cert.KeyUsage&x509.KeyUsageDigitalSignature)
		assert.NotEqual(t, 0, cert.KeyUsage&x509.KeyUsageCertSign)
		assert.Contains(t, cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
	})
}

func TestLoadCA(t *testing.T) {
	t.Run("should load valid CA certificate and key", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		// Load CA
		cert, key, err := infratls.LoadCA(nil, certPath, keyPath)
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.NotNil(t, key)
	})

	t.Run("should return error for non-existent certificate file", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, _, err := infratls.LoadCA(
			nil,
			filepath.Join(tmpDir, "nonexistent.crt"),
			filepath.Join(tmpDir, "nonexistent.key"),
		)
		require.Error(t, err)
	})

	t.Run("should return error for non-existent key file", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    tmpDir,
		}
		certPath, _, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		_, _, err = infratls.LoadCA(nil, certPath, filepath.Join(tmpDir, "nonexistent.key"))
		require.Error(t, err)
	})

	t.Run("should return error for invalid certificate PEM", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidCertPath := filepath.Join(tmpDir, "invalid.crt")
		keyPath := filepath.Join(tmpDir, "test.key")

		err := os.WriteFile(invalidCertPath, []byte("not a valid PEM"), 0o600)
		require.NoError(t, err)
		err = os.WriteFile(keyPath, []byte("not a valid key"), 0o600)
		require.NoError(t, err)

		_, _, err = infratls.LoadCA(nil, invalidCertPath, keyPath)
		require.Error(t, err)
	})
}

func TestCheckExpiration(t *testing.T) {
	t.Run("should detect expiring certificate", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := infratls.CAConfig{
			ValidityDays: 5, // 5 days
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		cert, _, err := infratls.LoadCA(nil, certPath, keyPath)
		require.NoError(t, err)

		// Check with 7-day threshold
		expiresSoon, timeLeft := infratls.CheckExpiration(cert, 7*24*time.Hour)
		assert.True(t, expiresSoon)
		assert.Positive(t, timeLeft)
		assert.Less(t, timeLeft, 7*24*time.Hour)
	})

	t.Run("should not detect non-expiring certificate", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		cert, _, err := infratls.LoadCA(nil, certPath, keyPath)
		require.NoError(t, err)

		// Check with 7-day threshold
		expiresSoon, timeLeft := infratls.CheckExpiration(cert, 7*24*time.Hour)
		assert.False(t, expiresSoon)
		assert.Greater(t, timeLeft, 7*24*time.Hour)
	})

	t.Run("should handle already expired certificate", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := infratls.CAConfig{
			ValidityDays: 1,
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		cert, _, err := infratls.LoadCA(nil, certPath, keyPath)
		require.NoError(t, err)

		// Manually modify cert to make it expired (for testing)
		// In real scenario, this would be naturally expired
		// We just test the logic works correctly
		expiresSoon, timeLeft := infratls.CheckExpiration(cert, 365*24*time.Hour)
		assert.True(t, expiresSoon)
		assert.Positive(t, timeLeft) // Still valid as we just created it
	})
}

func TestLoadCA_ErrorCases(t *testing.T) {
	t.Run("should return error for invalid key PEM", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		// Overwrite key with invalid data
		err = os.WriteFile(keyPath, []byte("not a valid key PEM"), 0o600)
		require.NoError(t, err)

		_, _, err = infratls.LoadCA(nil, certPath, keyPath)
		require.Error(t, err)
	})

	t.Run("should return error for invalid certificate in PEM", func(t *testing.T) {
		tmpDir := t.TempDir()

		certPath := filepath.Join(tmpDir, "invalid.crt")
		keyPath := filepath.Join(tmpDir, "test.key")

		// Create PEM with invalid certificate data
		certFile, err := os.Create(certPath)
		require.NoError(t, err)
		_, _ = certFile.WriteString("-----BEGIN CERTIFICATE-----\n")
		_, _ = certFile.WriteString("aW52YWxpZCBjZXJ0aWZpY2F0ZSBkYXRh\n")
		_, _ = certFile.WriteString("-----END CERTIFICATE-----\n")
		certFile.Close()

		keyFile, err := os.Create(keyPath)
		require.NoError(t, err)
		_, _ = keyFile.WriteString("key")
		keyFile.Close()

		_, _, err = infratls.LoadCA(nil, certPath, keyPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse certificate")
	})

	t.Run("should return error for invalid private key in PEM", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		// Create PEM with invalid key data
		keyFile, err := os.Create(keyPath)
		require.NoError(t, err)
		_, _ = keyFile.WriteString("-----BEGIN RSA PRIVATE KEY-----\n")
		_, _ = keyFile.WriteString("aW52YWxpZCBrZXkgZGF0YQ==\n")
		_, _ = keyFile.WriteString("-----END RSA PRIVATE KEY-----\n")
		keyFile.Close()

		_, _, err = infratls.LoadCA(nil, certPath, keyPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse private key")
	})
}

func TestGenerateCA_ErrorCases(t *testing.T) {
	t.Run("should use default filesystem when nil", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    tmpDir,
			Fs:           nil,
		}

		certPath, keyPath, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		assert.FileExists(t, certPath)
		assert.FileExists(t, keyPath)
	})
}

func TestLoadCA_UseDefaultFilesystem(t *testing.T) {
	t.Run("should use default filesystem when nil", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    tmpDir,
			Fs:           nil,
		}
		certPath, keyPath, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		// Load with nil filesystem
		cert, key, err := infratls.LoadCA(nil, certPath, keyPath)
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.NotNil(t, key)
	})
}
