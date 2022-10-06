package urlreplacer

import (
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/evg4b/uncors/pkg/urlx"
)

func NormaiseMappings(mappings map[string]string, httpPort, httpsPort int) (map[string]string, error) {
	processedMappings := map[string]string{}
	for source, target := range mappings {
		sourceURL, err := urlx.Parse(source)
		if err != nil {
			return nil, ErrInvalidSourceURL
		}

		if strings.EqualFold(sourceURL.Scheme, "http") || len(sourceURL.Scheme) == 0 {
			ss, err := assinPort(*sourceURL, "http", httpPort)
			if err != nil {
				return nil, err
			}

			processedMappings[ss] = target
		}

		if strings.EqualFold(sourceURL.Scheme, "https") || len(sourceURL.Scheme) == 0 {
			ss, err := assinPort(*sourceURL, "https", httpsPort)
			if err != nil {
				return nil, err
			}

			processedMappings[ss] = target
		}
	}

	return processedMappings, nil
}

func assinPort(u url.URL, scheme string, port int) (string, error) {
	host, _, err := urlx.SplitHostPort(&u)
	if err != nil {
		return "", err
	}

	u.Scheme = scheme
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))

	return u.String(), nil
}
