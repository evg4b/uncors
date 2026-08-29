package hosts

import (
	"strconv"

	"github.com/evg4b/uncors/internal/urlpattern"
)

type host string

func (h host) Host() urlpattern.Host {
	return urlpattern.Host{Hostname: string(h)}
}

func (h host) HTTP() urlpattern.Host {
	return urlpattern.Host{Scheme: "http", Hostname: string(h)}
}

func (h host) HTTPS() urlpattern.Host {
	return urlpattern.Host{Scheme: "https", Hostname: string(h)}
}

func (h host) Port(port int) urlpattern.Host {
	return urlpattern.Host{Hostname: string(h), Port: strconv.Itoa(port)}
}

func (h host) HTTPSPort(port int) urlpattern.Host {
	return urlpattern.Host{Scheme: "https", Hostname: string(h), Port: strconv.Itoa(port)}
}

func (h host) HTTPPort(port int) urlpattern.Host {
	return urlpattern.Host{Scheme: "http", Hostname: string(h), Port: strconv.Itoa(port)}
}

func (h host) Scheme(scheme string) urlpattern.Host {
	return urlpattern.Host{Scheme: scheme, Hostname: string(h)}
}

func (h host) NoScheme() urlpattern.Host {
	return urlpattern.Host{Hostname: string(h)}
}

// Parse builds a urlpattern.Host from a raw string. It is intended for tests that
// need to construct a mapping host from a literal. Invalid input (including an
// empty string) yields the zero Host, which is convenient for exercising
// validation paths.
func Parse(raw string) urlpattern.Host {
	parsed, err := urlpattern.Parse(raw)
	if err != nil {
		return urlpattern.Host{}
	}

	return *parsed
}
