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
const defaultHttpPort = 80

const httpsScheme = "https"
const defaultHttpsPort = 443

func NormaliseMappings(mappings map[string]string, httpPort, httpsPort int, useHttps bool) (map[string]string, error) {
	processedMappings := map[string]string{}
	for source, target := range mappings {
		sourceURL, err := urlx.Parse(source)
		if err != nil {
			return nil, ErrInvalidSourceURL
		}

		if strings.EqualFold(sourceURL.Scheme, httpScheme) || len(sourceURL.Scheme) == 0 {
			normalisedSource, err := assignPortAndScheme(*sourceURL, httpScheme, httpPort)
			if err != nil {
				return nil, err
			}

			processedMappings[normalisedSource] = target
		}

		if useHttps && (strings.EqualFold(sourceURL.Scheme, httpsScheme) || len(sourceURL.Scheme) == 0) {
			normalisedSource, err := assignPortAndScheme(*sourceURL, httpsScheme, httpsPort)
			if err != nil {
				return nil, err
			}

			processedMappings[normalisedSource] = target
		}
	}

	return processedMappings, nil
}

func assignPortAndScheme(u url.URL, scheme string, port int) (string, error) {
	host, _, err := urlx.SplitHostPort(&u)
	if err != nil {
		return "", fmt.Errorf("failed split host and port: %v", err)
	}

	u.Scheme = scheme
	if !(isDefaultPort(scheme, port)) {
		u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	} else {
		u.Host = host
	}

	return u.String(), nil
}

func isDefaultPort(scheme string, port int) bool {
	return strings.EqualFold(httpScheme, scheme) && port == defaultHttpPort ||
		strings.EqualFold(httpsScheme, scheme) && port == defaultHttpsPort
}
