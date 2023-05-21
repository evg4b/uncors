package proxy

import (
	"net/http"

	"github.com/evg4b/uncors/internal/infra"
)

func (m *Handler) makeOptionsResponse(writer http.ResponseWriter, req *http.Request) error {
	infra.WriteCorsHeaders(writer.Header())
	m.logger.PrintResponse(&http.Response{
		StatusCode: http.StatusOK,
		Request:    req,
	})

	return nil
}
