package urlt

import (
	"errors"
	"strings"
)

type Host struct {
	Scheme   string
	Hostname string
	Port     string
}

func (h *Host) String() string {
	if h.Port != "" {
		return h.Hostname + ":" + h.Port
	}
	return h.Hostname
}

func ParseHost(rawURL string) (*Host, error) {
	var rest string
	var err error

	if rawURL == "" {
		return nil, errors.New("empty host")
	}

	if stringContainsCTLByte(rawURL) {
		return nil, errors.New("net/url: invalid control character in URL")
	}

	if !strings.Contains(rawURL, "//") {
		rawURL = "//" + rawURL
	}

	host := new(Host)

	if host.Scheme, rest, err = getScheme(rawURL); err != nil {
		return nil, err
	}

	host.Scheme = strings.ToLower(host.Scheme)

	rest = strings.TrimPrefix(rest, "//")

	if i := strings.Index(rest, "/"); i >= 0 {
		return nil, errors.New("host must not contain a path")
	}

	hostWithPort, err := parseHost(host.Scheme, rest)
	if err != nil {
		return nil, err
	}

	hostName, port := splitHostPort(hostWithPort)

	host.Hostname = hostName
	host.Port = port

	return host, nil
}
