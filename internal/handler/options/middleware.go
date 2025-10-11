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
	code    int
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
	infra.WriteCorsHeadersForOptions(resp.Header(), req.Header)

	for key, value := range m.headers {
		resp.Header().Set(key, value)
	}

	statucCode := helpers.NormaliseStatucCode(m.code)
	resp.WriteHeader(statucCode)

	tui.PrintResponse(m.logger, req, statucCode)
}
