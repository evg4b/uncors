package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

const (
	keySize             = 2048
	serialNumberBits    = 128
	dirPermissions      = 0o755
	keyFilePermissions  = 0o600
	defaultOrganization = "UNCORS Development CA"
	defaultCommonName   = "UNCORS Local Development Root CA"
	defaultCountry      = "US"
)

// CAConfig represents configuration for CA certificate generation.
type CAConfig struct {
	ValidityDays int
	OutputDir    string
}

// GenerateCA generates a new CA certificate and private key.
// Returns the paths to the generated certificate and key files.
func GenerateCA(config CAConfig) (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.AddDate(0, 0, config.ValidityDays)

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), serialNumberBits))
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{defaultOrganization},
			CommonName:   defaultCommonName,
			Country:      []string{defaultCountry},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(config.OutputDir, dirPermissions); err != nil {
		return "", "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write certificate file
	certPath := filepath.Join(config.OutputDir, "ca.crt")
	certFile, err := os.Create(certPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate file: %w", err)
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}); err != nil {
		return "", "", fmt.Errorf("failed to write certificate: %w", err)
	}

	// Write private key file
	keyPath := filepath.Join(config.OutputDir, "ca.key")
	keyFile, err := os.Create(keyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyFile.Close()

	if err := os.Chmod(keyPath, keyFilePermissions); err != nil {
		return "", "", fmt.Errorf("failed to set key file permissions: %w", err)
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	if err := pem.Encode(keyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}); err != nil {
		return "", "", fmt.Errorf("failed to write private key: %w", err)
	}

	return certPath, keyPath, nil
}

// LoadCA loads CA certificate and private key from files.
func LoadCA(certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil || certBlock.Type != "CERTIFICATE" {
		return nil, nil, ErrInvalidCertificatePEM
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read key file: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil || keyBlock.Type != "RSA PRIVATE KEY" {
		return nil, nil, ErrInvalidPrivateKeyPEM
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return cert, privateKey, nil
}

// CheckExpiration checks if certificate expires within the given duration.
// Returns true if certificate expires soon, along with the time until expiration.
func CheckExpiration(cert *x509.Certificate, threshold time.Duration) (bool, time.Duration) {
	timeLeft := time.Until(cert.NotAfter)
	expiresSoon := timeLeft < threshold

	return expiresSoon, timeLeft
}
