package proxy

import (
	"net/http"
	"net/url"

	"github.com/evg4b/uncors/internal/urlreplacer"
)

type URLReplacerFactory interface {
	Make(requestURL *url.URL) (*urlreplacer.Replacer, *urlreplacer.Replacer, error)
}

type HandlerOption = func(*Handler)

func WithURLReplacerFactory(replacerFactory URLReplacerFactory) HandlerOption {
	return func(pm *Handler) {
		pm.replacerFactory = replacerFactory
	}
}

func WithHTTPClient(http *http.Client) HandlerOption {
	return func(pm *Handler) {
		pm.http = http
	}
}
