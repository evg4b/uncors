package tls

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
)

// GetCAPath returns the default path to the CA certificate directory.
func GetCAPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "uncors"), nil
}

// CAExists checks if CA certificate files exist.
func CAExists() bool {
	caDir, err := GetCAPath()
	if err != nil {
		return false
	}

	certPath := filepath.Join(caDir, "ca.crt")
	keyPath := filepath.Join(caDir, "ca.key")

	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	return certErr == nil && keyErr == nil
}

// LoadDefaultCA loads the CA certificate from the default location.
func LoadDefaultCA() (*x509.Certificate, *rsa.PrivateKey, error) {
	caDir, err := GetCAPath()
	if err != nil {
		return nil, nil, err
	}

	certPath := filepath.Join(caDir, "ca.crt")
	keyPath := filepath.Join(caDir, "ca.key")

	return LoadCA(certPath, keyPath)
}
