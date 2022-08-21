package server

import "github.com/evg4b/uncors/internal/processor"

type serverOption = func(s *Server)

func WithHTTPPort(port int) serverOption {
	return func(s *Server) {
		s.httpPort = port
	}
}

func WithHTTPSPort(port int) serverOption {
	return func(s *Server) {
		s.httpPort = port
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
		s.handler = requestProcessor.HandleRequest
	}
}
