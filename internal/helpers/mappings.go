package helpers

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/evg4b/uncors/internal/configuration"

	"github.com/evg4b/uncors/pkg/urlx"
)

const (
	httpScheme      = "http"
	defaultHTTPPort = 80
)

const (
	httpsScheme      = "https"
	defaultHTTPSPort = 443
)

func NormaliseMappings(
	mappings []configuration.URLMapping,
	httpPort,
	httpsPort int,
	useHTTPS bool,
) (map[string]string, error) {
	processedMappings := map[string]string{}
	for _, mapping := range mappings {
		sourceURL, err := urlx.Parse(mapping.From)
		if err != nil {
			return nil, fmt.Errorf("failed to parse source url: %w", err)
		}

		if isApplicableScheme(sourceURL.Scheme, httpScheme) {
			normalisedSource := assignPortAndScheme(*sourceURL, httpScheme, httpPort)
			processedMappings[normalisedSource] = mapping.To
		}

		if useHTTPS && isApplicableScheme(sourceURL.Scheme, httpsScheme) {
			normalisedSource := assignPortAndScheme(*sourceURL, httpsScheme, httpsPort)
			processedMappings[normalisedSource] = mapping.To
		}
	}

	return processedMappings, nil
}

func assignPortAndScheme(parsedURL url.URL, scheme string, port int) string {
	host, _, _ := urlx.SplitHostPort(&parsedURL)
	parsedURL.Scheme = scheme

	if !(isDefaultPort(scheme, port)) {
		parsedURL.Host = net.JoinHostPort(host, strconv.Itoa(port))
	} else {
		parsedURL.Host = host
	}

	return parsedURL.String()
}

func isDefaultPort(scheme string, port int) bool {
	return strings.EqualFold(httpScheme, scheme) && port == defaultHTTPPort ||
		strings.EqualFold(httpsScheme, scheme) && port == defaultHTTPSPort
}

func isApplicableScheme(scheme, expectedScheme string) bool {
	return strings.EqualFold(scheme, expectedScheme) || len(scheme) == 0
}