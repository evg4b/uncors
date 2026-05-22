package tls

import "errors"

var (
	// ErrInvalidCertificatePEM is returned when certificate PEM cannot be decoded.
	ErrInvalidCertificatePEM = errors.New("failed to decode certificate PEM")

	// ErrInvalidPrivateKeyPEM is returned when private key PEM cannot be decoded.
	ErrInvalidPrivateKeyPEM = errors.New("failed to decode private key PEM")

	// ErrNoCertificateAvailable is returned when no certificate is available and auto-generation is disabled.
	ErrNoCertificateAvailable = errors.New("no certificate available and auto-generation is disabled")

	// ErrNoMappingsProvided is returned when no mappings are provided for TLS config.
	ErrNoMappingsProvided = errors.New("no mappings provided for TLS config")

	// ErrNoSNIProvided is returned when no SNI is provided in the client hello.
	ErrNoSNIProvided = errors.New("no SNI provided in client hello")

	// ErrNoCertificateForHost is returned when no certificate is available for the requested host.
	ErrNoCertificateForHost = errors.New("no certificate available for host")

	// ErrCACertExpired is returned when the CA certificate has already expired.
	ErrCACertExpired = errors.New("CA certificate has expired! Please regenerate it with: uncors generate-certs --force")

	// ErrCACertExpiringSoon is returned when the CA certificate is close to expiring.
	ErrCACertExpiringSoon = errors.New("consider regenerating with: uncors generate-certs --force")
)
