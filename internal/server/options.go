package server

import (
	"net"
	"net/http"
	"strconv"

	"github.com/evg4b/uncors/internal/processor"
)

type Option = func(s *Server)

func WithHTTP(baseAddress string, port int) Option {
	address := net.JoinHostPort(baseAddress, strconv.Itoa(port))

	return func(server *Server) {
		server.http = &http.Server{
			ReadHeaderTimeout: readHeaderTimeout,
			Addr:              address,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				server.handler.ServeHTTP(w, r)
			}),
		}
	}
}

func WithHTTPListener(listener Listener) Option {
	return func(server *Server) {
		server.http = listener
	}
}

func WithHTTPS(baseAddress string, port int) Option {
	address := net.JoinHostPort(baseAddress, strconv.Itoa(port))

	return func(server *Server) {
		server.https = &http.Server{
			ReadHeaderTimeout: readHeaderTimeout,
			Addr:              address,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				server.handler.ServeHTTP(w, r)
			}),
		}
	}
}

func WithHTTPSListener(listener Listener) Option {
	return func(server *Server) {
		server.https = listener
	}
}

func WithSslCert(cert string) Option {
	return func(s *Server) {
		s.cert = cert
	}
}

func WithSslKey(key string) Option {
	return func(s *Server) {
		s.key = key
	}
}

func WithRequestProcessor(requestProcessor *processor.RequestProcessor) Option {
	return func(s *Server) {
		s.handler = requestProcessor
	}
}
