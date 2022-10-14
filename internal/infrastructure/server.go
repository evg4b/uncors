package infrastructure

import (
	"net"
	"net/http"
	"strconv"
	"time"
)

const readHeaderTimeout = 30 * time.Second

func NewServer(addr string, port int, handler http.Handler) *http.Server {
	return &http.Server{
		ReadHeaderTimeout: readHeaderTimeout,
		Handler:           handler,
		Addr:              net.JoinHostPort(addr, strconv.Itoa(port)),
	}
}
