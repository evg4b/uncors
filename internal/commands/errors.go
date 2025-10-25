package commands

import "errors"

var (
	// ErrCAAlreadyExists is returned when CA certificate already exists.
	ErrCAAlreadyExists = errors.New("CA certificate already exists")

	// ErrCAKeyAlreadyExists is returned when CA private key already exists.
	ErrCAKeyAlreadyExists = errors.New("CA private key already exists")
)
