package server

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

const (
	CACertFileName = "ca.crt"
	CAKeyFileName  = "ca.key"
)

// GetCAPath returns the directory the local CA lives in.
//
// It follows $XDG_CONFIG_HOME before the home directory, so that a process
// without a usable home — a container running as a numeric user, for instance —
// still has somewhere to keep it.
func GetCAPath() (string, error) {
	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		return filepath.Join(configDir, "uncors"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to locate the CA directory (set --ca-dir or $XDG_CONFIG_HOME): %w", err)
	}

	return filepath.Join(homeDir, ".config", "uncors"), nil
}

// CAPathOr returns the configured CA directory, falling back to the default.
func CAPathOr(caDir string) (string, error) {
	if caDir != "" {
		return caDir, nil
	}

	return GetCAPath()
}

// CAExists checks if CA certificate files exist in the given directory ("" for
// the default one).
func CAExists(fs afero.Fs, caDir string) bool {
	resolved, err := CAPathOr(caDir)
	if err != nil {
		return false
	}

	certPath := filepath.Join(resolved, CACertFileName)
	keyPath := filepath.Join(resolved, CAKeyFileName)

	_, certErr := fs.Stat(certPath)
	_, keyErr := fs.Stat(keyPath)

	return certErr == nil && keyErr == nil
}

// LoadDefaultCA loads the CA certificate from the given directory ("" for the
// default one).
func LoadDefaultCA(fs afero.Fs, caDir string) (*x509.Certificate, *rsa.PrivateKey, error) {
	resolved, err := CAPathOr(caDir)
	if err != nil {
		return nil, nil, err
	}

	certPath := filepath.Join(resolved, CACertFileName)
	keyPath := filepath.Join(resolved, CAKeyFileName)

	return LoadCA(fs, certPath, keyPath)
}
