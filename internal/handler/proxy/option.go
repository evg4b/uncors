package proxy

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/infra"
)

func (h *Handler) makeOptionsResponse(writer http.ResponseWriter, req *http.Request) {
	log.With("method", req.Method).
		With("url", req.URL).
		Debug("Handle options request")

	infra.WriteCorsHeaders(writer.Header())
}
