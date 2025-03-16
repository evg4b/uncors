package options

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
)

type Middleware struct {
	logger  contracts.Logger
	headers map[string]string
	code    uint
}

func NewMiddleware(options ...MiddlewareOption) *Middleware {
	return helpers.ApplyOptions(&Middleware{}, options)
}

func (m *Middleware) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(resp contracts.ResponseWriter, req *contracts.Request) {
		if strings.EqualFold(req.Method, http.MethodOptions) {
			m.handle(resp, req)
		} else {
			next.ServeHTTP(resp, req)
		}
	})
}

func (m *Middleware) handle(resp http.ResponseWriter, req *http.Request) {
	infra.WriteCorsHeaders(resp.Header())
	tui.PrintResponse(m.logger, req, http.StatusOK)
}
