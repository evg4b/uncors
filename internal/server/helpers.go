package server

import (
	"errors"
	"net/http"

	"github.com/hashicorp/go-multierror"
)

func (s *Server) isHTTPSAvailable() bool {
	return len(s.cert) > 0 && len(s.key) > 0
}

func isSuccessClosed(err error) bool {
	if err == nil {
		return true
	}

	if merr, ok := err.(*multierror.Error); ok { // nolint: errorlint
		for _, wrappedError := range merr.Errors {
			if !errors.Is(wrappedError, http.ErrServerClosed) {
				return false
			}
		}

		return true
	}

	return errors.Is(err, http.ErrServerClosed)
}
