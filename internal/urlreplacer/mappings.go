package urlreplacer

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/evg4b/uncors/pkg/urlx"
)

const httpScheme = "http"
const defaultHTTPPort = 80

const httpsScheme = "https"
const defaultHTTPSPort = 443

func NormaliseMappings(mappings map[string]string, httpPort, httpsPort int, useHTTPS bool) (map[string]string, error) {
	processedMappings := map[string]string{}
	for source, target := range mappings {
		sourceURL, err := urlx.Parse(source)
		if err != nil {
			return nil, ErrInvalidSourceURL
		}

		if isApplicableScheme(sourceURL.Scheme, httpScheme) {
			normalisedSource, err := assignPortAndScheme(*sourceURL, httpScheme, httpPort)
			if err != nil {
				return nil, err
			}

			processedMappings[normalisedSource] = target
		}

		if useHTTPS && isApplicableScheme(sourceURL.Scheme, httpsScheme) {
			normalisedSource, err := assignPortAndScheme(*sourceURL, httpsScheme, httpsPort)
			if err != nil {
				return nil, err
			}

			processedMappings[normalisedSource] = target
		}
	}

	return processedMappings, nil
}

func assignPortAndScheme(parsedURL url.URL, scheme string, port int) (string, error) {
	host, _, err := urlx.SplitHostPort(&parsedURL)
	if err != nil {
		return "", fmt.Errorf("failed split host and port: %w", err)
	}

	parsedURL.Scheme = scheme
	if !(isDefaultPort(scheme, port)) {
		parsedURL.Host = net.JoinHostPort(host, strconv.Itoa(port))
	} else {
		parsedURL.Host = host
	}

	return parsedURL.String(), nil
}

func isDefaultPort(scheme string, port int) bool {
	return strings.EqualFold(httpScheme, scheme) && port == defaultHTTPPort ||
		strings.EqualFold(httpsScheme, scheme) && port == defaultHTTPSPort
}

func isApplicableScheme(scheme, expectedScheme string) bool {
	return strings.EqualFold(scheme, expectedScheme) || len(scheme) == 0
}
