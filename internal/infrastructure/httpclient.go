package infrastructure

import (
	"crypto/tls"
	"net/http"
	"time"
)

const defaulTimeout = 5 * time.Minute

var HTTPCLient = http.Client{
	CheckRedirect: func(r *http.Request, v []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		},
	},
	Jar:     nil,
	Timeout: defaulTimeout,
}
