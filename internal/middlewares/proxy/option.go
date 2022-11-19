package proxy

import (
	"net/http"

	"github.com/evg4b/uncors/internal/infrastructure"
)

func (m *Middleware) makeOptionsResponse(writer http.ResponseWriter, req *http.Request) error {
	infrastructure.WriteCorsHeaders(writer.Header())
	m.logger.PrintResponse(&http.Response{
		StatusCode: http.StatusOK,
		Request:    req,
	})

	return nil
}
