package options

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
)

type Middleware struct {
	headers map[string]string
	code    int
}

func NewMiddleware(options ...MiddlewareOption) *Middleware {
	return helpers.ApplyOptions(&Middleware{}, options)
}

func (m *Middleware) ServeHTTP(resp contracts.ResponseWriter, req *contracts.Request, next contracts.Next) error {
	if strings.EqualFold(req.Method, http.MethodOptions) {
		m.handle(resp, req)

		return nil
	}

	return next(resp, req)
}

func (m *Middleware) handle(resp http.ResponseWriter, req *http.Request) {
	infra.WriteCorsHeadersForOptions(resp.Header(), req.Header)

	for key, value := range m.headers {
		resp.Header().Set(key, value)
	}

	resp.WriteHeader(helpers.NormaliseStatusCode(m.code))
}

type MiddlewareOption = func(*Middleware)

func WithHeaders(headers map[string]string) MiddlewareOption {
	return func(m *Middleware) {
		m.headers = headers
	}
}

func WithCode(code int) MiddlewareOption {
	return func(m *Middleware) {
		m.code = code
	}
}
