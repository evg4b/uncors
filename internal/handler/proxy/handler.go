package proxy

import (
	"fmt"
	"io"
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/go-http-utils/headers"
)

type Handler struct {
	replacers     urlreplacer.ReplacerFactory
	http          contracts.HTTPClient
	proxyLogger   contracts.Logger
	rewriteLogger contracts.Logger
}

func NewProxyHandler(options ...HandlerOption) *Handler {
	middleware := helpers.ApplyOptions(&Handler{}, options)

	helpers.AssertIsDefined(middleware.replacers, "ProxyHandler: ReplacerFactory is not configured")
	helpers.AssertIsDefined(middleware.proxyLogger, "ProxyHandler: Logger is not configured")
	helpers.AssertIsDefined(middleware.rewriteLogger, "ProxyHandler: Logger is not configured")
	helpers.AssertIsDefined(middleware.http, "ProxyHandler: Http client is not configured")

	return middleware
}

func (h *Handler) ServeHTTP(response contracts.ResponseWriter, request *contracts.Request) {
	err := h.handle(response, request)
	if err != nil {
		h.proxyLogger.Error("Proxy handler error", "error", err, "url", request.URL.String())
		infra.HTTPError(response, err)
	}
}

func (h *Handler) handle(resp http.ResponseWriter, req *http.Request) error {
	targetReplacer, sourceReplacer, err := h.createReplacers(req)
	if err != nil {
		return err
	}

	originalRequest, err := h.makeOriginalRequest(req, targetReplacer)
	if err != nil {
		return fmt.Errorf("failed to create request to original source: %w", err)
	}

	originalResponse, err := h.executeQuery(originalRequest)
	if err != nil {
		return err
	}

	defer helpers.CloseSafe(originalResponse.Body)

	err = h.makeUncorsResponse(originalResponse, resp, sourceReplacer, req)
	if err != nil {
		return fmt.Errorf("failed to make uncors response: %w", err)
	}

	return nil
}

func (h *Handler) makeOriginalRequest(
	req *http.Request,
	replacer *urlreplacer.Replacer,
) (*http.Request, error) {
	url, err := replacer.Replace(req.URL.String())
	if err != nil {
		return nil, fmt.Errorf("failed to replace URL: %w", err)
	}

	originalRequest, err := http.NewRequestWithContext(req.Context(), req.Method, url, req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to original server: %w", err)
	}

	err = copyHeaders(req.Header, originalRequest.Header, modificationsMap{
		headers.Origin:  replacer.Replace,
		headers.Referer: replacer.Replace,
	})
	if err != nil {
		return nil, err
	}

	copyCookiesToTarget(req, replacer, originalRequest)

	return originalRequest, nil
}

func (h *Handler) makeUncorsResponse(
	original *http.Response,
	target http.ResponseWriter,
	replacer *urlreplacer.Replacer,
	req *http.Request,
) error {
	copyCookiesToSource(original, replacer, target)

	err := copyHeaders(original.Header, target.Header(), modificationsMap{
		headers.Location: func(s string) (string, error) { //nolint:unparam
			return replacer.ReplaceSoft(s), nil
		},
	})
	if err != nil {
		return err
	}

	origin := req.Header.Get(headers.Origin)
	infra.WriteCorsHeaders(target.Header(), origin)

	target.WriteHeader(original.StatusCode)

	_, err = io.Copy(target, original.Body)
	if err != nil {
		return fmt.Errorf("failed to copy body to response: %w", err)
	}

	return nil
}

func (h *Handler) createReplacers(req *http.Request) (*urlreplacer.Replacer, *urlreplacer.Replacer, error) {
	rewriteHost, err := rewrite.GetRewriteHost(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get rewrite host: %w", err)
	}

	if rewriteHost == "" {
		targetReplacer, sourceReplacer, err := h.replacers.Make(req.URL)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to transform general url: %w", err)
		}

		return targetReplacer, sourceReplacer, nil
	}

	target, err := urlreplacer.NewReplacer(req.URL.Host, rewriteHost)
	if err != nil {
		return nil, nil, err
	}

	source, err := urlreplacer.NewReplacer(rewriteHost, req.URL.Host)
	if err != nil {
		return nil, nil, err
	}

	return target, source, nil
}

func (h *Handler) executeQuery(request *http.Request) (*http.Response, error) {
	originalResponse, err := h.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	tui.PrintResponse(h.logger(request), originalResponse.Request, originalResponse.StatusCode)

	return originalResponse, nil
}

func (h *Handler) logger(request *http.Request) contracts.Logger {
	if rewrite.IsRewriteRequest(request) {
		return h.rewriteLogger
	}

	return h.proxyLogger
}

type HandlerOption = func(*Handler)

func WithURLReplacerFactory(replacerFactory urlreplacer.ReplacerFactory) HandlerOption {
	return func(m *Handler) {
		m.replacers = replacerFactory
	}
}

func WithHTTPClient(http contracts.HTTPClient) HandlerOption {
	return func(m *Handler) {
		m.http = http
	}
}

func WithProxyLogger(logger contracts.Logger) HandlerOption {
	return func(m *Handler) {
		m.proxyLogger = logger
	}
}

func WithRewriteLogger(logger contracts.Logger) HandlerOption {
	return func(m *Handler) {
		m.rewriteLogger = logger
	}
}
