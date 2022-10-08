package infrastructure

import (
	"crypto/tls"
	"net/http"
	"time"
)

const defaultTimeout = 5 * time.Minute

var HTTPClient = http.Client{
	CheckRedirect: func(r *http.Request, v []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		},
		Proxy: http.ProxyFromEnvironment,
	},
	Jar:     nil,
	Timeout: defaultTimeout,
}
