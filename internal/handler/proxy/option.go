package proxy

import (
	"net/http"

	"github.com/evg4b/uncors/internal/infra"
)

func (h *Handler) makeOptionsResponse(writer http.ResponseWriter, req *http.Request) error {
	infra.WriteCorsHeaders(writer.Header())
	h.logger.PrintResponse(req, http.StatusOK)

	return nil
}
