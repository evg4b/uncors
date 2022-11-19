package urlreplacer

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

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

func NormaliseMappings(mappings map[string]string, httpPort, httpsPort int, useHTTPS bool) (map[string]string, error) {
	processedMappings := map[string]string{}
	for source, target := range mappings {
		sourceURL, err := urlx.Parse(source)
		if err != nil {
			return nil, fmt.Errorf("failed to parse source url: %w", err)
		}

		if isApplicableScheme(sourceURL.Scheme, httpScheme) {
			normalisedSource := assignPortAndScheme(*sourceURL, httpScheme, httpPort)
			processedMappings[normalisedSource] = target
		}

		if useHTTPS && isApplicableScheme(sourceURL.Scheme, httpsScheme) {
			normalisedSource := assignPortAndScheme(*sourceURL, httpsScheme, httpsPort)
			processedMappings[normalisedSource] = target
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
