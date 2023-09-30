package hosts

import (
	"strconv"
)

type host string

func (h host) Host() string {
	return string(h)
}

func (h host) HTTP() string {
	return "http://" + string(h)
}

func (h host) HTTPS() string {
	return "https://" + string(h)
}

func (h host) Port(port int) string {
	return h.jonPort(h.Host(), port)
}

func (h host) HTTPSPort(port int) string {
	return h.jonPort(h.HTTPS(), port)
}

func (h host) HTTPPort(port int) string {
	return h.jonPort(h.HTTP(), port)
}

func (h host) jonPort(host string, port int) string {
	return host + ":" + strconv.Itoa(port)
}
