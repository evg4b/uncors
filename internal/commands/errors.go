package commands

import "errors"

var (
	ErrCAAlreadyExists = errors.New("CA certificate already exists")

	ErrCAKeyAlreadyExists = errors.New("CA private key already exists")
)
