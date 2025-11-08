package infra

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evg4b/uncors/internal/urlparser"
)

const defaultTimeout = 5 * time.Minute

func MakeHTTPClient(proxy string) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	if proxy != "" {
		parsedURL, err := urlparser.Parse(proxy)
		if err != nil {
			panic(fmt.Errorf("failed to create http client: %w", err))
		}

		transport.Proxy = http.ProxyURL(parsedURL)
	}

	return &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: transport,
		Timeout:   defaultTimeout,
	}
}
