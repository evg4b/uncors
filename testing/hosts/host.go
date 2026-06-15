package hosts

import (
	"strconv"

	"github.com/evg4b/uncors/pkg/urlt"
)

type host string

func (h host) Host() urlt.Host {
	return urlt.Host{Hostname: string(h)}
}

func (h host) HTTP() urlt.Host {
	return urlt.Host{Scheme: "http", Hostname: string(h)}
}

func (h host) HTTPS() urlt.Host {
	return urlt.Host{Scheme: "https", Hostname: string(h)}
}

func (h host) Port(port int) urlt.Host {
	return urlt.Host{Hostname: string(h), Port: strconv.Itoa(port)}
}

func (h host) HTTPSPort(port int) urlt.Host {
	return urlt.Host{Scheme: "https", Hostname: string(h), Port: strconv.Itoa(port)}
}

func (h host) HTTPPort(port int) urlt.Host {
	return urlt.Host{Scheme: "http", Hostname: string(h), Port: strconv.Itoa(port)}
}

func (h host) Scheme(scheme string) urlt.Host {
	return urlt.Host{Scheme: scheme, Hostname: string(h)}
}

func (h host) NoScheme() urlt.Host {
	return urlt.Host{Hostname: string(h)}
}

// Parse builds a urlt.Host from a raw string. It is intended for tests that
// need to construct a mapping host from a literal. Invalid input (including an
// empty string) yields the zero Host, which is convenient for exercising
// validation paths.
func Parse(raw string) urlt.Host {
	parsed, err := urlt.ParseHost(raw)
	if err != nil {
		return urlt.Host{}
	}

	return *parsed
}
