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

	// ErrNoSNIAndNoFallback is returned when no SNI is provided and no fallback host is available.
	ErrNoSNIAndNoFallback = errors.New("no SNI provided and no fallback host available")

	// ErrNoCertificateForHost is returned when no certificate is available for the requested host.
	ErrNoCertificateForHost = errors.New("no certificate available for host")
)
