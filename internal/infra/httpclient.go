package infra

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/evg4b/uncors/internal/telemetry"
)

const defaultTimeout = 5 * time.Minute

func MakeHTTPClient(proxy string) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	if proxy != "" {
		parsedURL, err := url.Parse(proxy)
		if err != nil {
			panic(fmt.Errorf("failed to create http client: %w", err))
		}

		transport.Proxy = http.ProxyURL(parsedURL)
	}

	return &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: telemetry.WrapTransport(transport),
		Timeout:   defaultTimeout,
	}
}
