package infra

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evg4b/uncors/pkg/urlx"
)

const defaultTimeout = 5 * time.Minute

var defaultHTTPClient = http.Client{
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	},
	Jar:     nil,
	Timeout: defaultTimeout,
}

func MakeHTTPClient(proxy string) *http.Client {
	if len(proxy) > 0 {
		parsedURL, err := urlx.Parse(proxy)
		if err != nil {
			panic(fmt.Errorf("failed to create http client: %w", err))
		}

		httpClient := defaultHTTPClient
		httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(parsedURL),
		}

		return &httpClient
	}

	return &defaultHTTPClient
}
