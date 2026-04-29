package options

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
)

type Middleware struct {
	output  contracts.Output
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

	statucCode := helpers.NormaliseStatusCode(m.code)
	resp.WriteHeader(statucCode)

	m.output.Request(helpers.ToRequestData(req, statucCode))
}

type MiddlewareOption = func(*Middleware)

func WithOutput(output contracts.Output) MiddlewareOption {
	return func(m *Middleware) {
		m.output = output
	}
}

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
