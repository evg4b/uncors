package server

import (
	"net"
	"net/http"
	"strconv"

	"github.com/evg4b/uncors/internal/processor"
)

type serverOption = func(s *Server)

func WithHTTP(baseAddress string, port int) serverOption {
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

func WithHTTPListner(listner Listner) serverOption {
	return func(server *Server) {
		server.http = listner
	}
}

func WithHTTPS(baseAddress string, port int) serverOption {
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

func WithHTTPSListner(listner Listner) serverOption {
	return func(server *Server) {
		server.https = listner
	}
}

func WithSslCert(cert string) serverOption {
	return func(s *Server) {
		s.cert = cert
	}
}

func WithSslKey(key string) serverOption {
	return func(s *Server) {
		s.key = key
	}
}

func WithRequstPropceessor(requestProcessor *processor.RequestProcessor) serverOption {
	return func(s *Server) {
		s.handler = requestProcessor
	}
}
