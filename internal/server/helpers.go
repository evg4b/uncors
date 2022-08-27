package server

import (
	"errors"
	"net/http"
)

func (s *Server) isHTTPSAvialable() bool {
	return len(s.cert) > 0 && len(s.key) > 0
}

func isSucessClosed(err error) bool {
	return err == nil || errors.Is(err, http.ErrServerClosed)
}
