package proxy

import (
	"net/http"

	"github.com/evg4b/uncors/internal/tui"

	"github.com/evg4b/uncors/internal/infra"
)

func (h *Handler) makeOptionsResponse(writer http.ResponseWriter, req *http.Request) {
	infra.WriteCorsHeaders(writer.Header())
	tui.PrintResponse(h.logger(req), req, http.StatusOK)
}
