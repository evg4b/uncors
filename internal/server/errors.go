package server

import "errors"

var (
	ErrInvalidCertificatePEM = errors.New("failed to decode certificate PEM")

	ErrInvalidPrivateKeyPEM = errors.New("failed to decode private key PEM")

	ErrNoSNIProvided = errors.New("no SNI provided in client hello")

	ErrNoCertificateForHost = errors.New("no certificate available for host")

	ErrCACertExpired = errors.New("CA certificate has expired! Please regenerate it with: uncors generate-certs --force")

	ErrCACertExpiringSoon = errors.New("consider regenerating with: uncors generate-certs --force")
)
