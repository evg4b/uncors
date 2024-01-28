package proxy

import (
	"net/http"

	"github.com/evg4b/uncors/internal/infra"
)

func (h *Handler) makeOptionsResponse(writer http.ResponseWriter, _ *http.Request) error {
	infra.WriteCorsHeaders(writer.Header())

	return nil
}
