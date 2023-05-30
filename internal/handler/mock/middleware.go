package mock

import (
	"net/http"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/spf13/afero"
)

type Middleware struct {
	response config.Response
	logger   contracts.Logger
	fs       afero.Fs
	after    func(duration time.Duration) <-chan time.Time
}

func NewMockMiddleware(options ...MiddlewareOption) *Middleware {
	middleware := &Middleware{}

	for _, option := range options {
		option(middleware)
	}

	return middleware
}

func (m *Middleware) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	response := m.response
	header := writer.Header()

	if response.Delay > 0 {
		m.logger.Debugf("Delay %s for %s", response.Delay, request.URL.RequestURI())
		ctx := request.Context()
		select {
		case <-ctx.Done():
			writer.WriteHeader(http.StatusServiceUnavailable)
			m.logger.Debugf("Delay for %s canceled", request.URL.RequestURI())

			return
		case <-m.after(response.Delay):
		}
	}

	infra.WriteCorsHeaders(header)
	for key, value := range response.Headers {
		header.Set(key, value)
	}

	if len(m.response.File) > 0 {
		err := m.serveFileContent(writer, request)
		if err != nil {
			infra.HTTPError(writer, err)

			return
		}
	} else {
		m.serveRawContent(writer)
	}

	m.logger.PrintResponse(&http.Response{
		Request:    request,
		StatusCode: response.Code,
	})
}

func normaliseCode(code int) int {
	if code == 0 {
		return http.StatusOK
	}

	return code
}
