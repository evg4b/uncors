package proxy

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
)

type RwreiteHandler struct {
	rewrite config.RewritingOption
	http    contracts.HTTPClient
	logger  contracts.Logger
}

func NewRwreiteHandler(options ...RewriteOption) *RwreiteHandler {
	return helpers.ApplyOptions(&RwreiteHandler{}, options)
}

func (h *RwreiteHandler) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) {
		next.ServeHTTP(writer, request)
	})
}

func (h *RwreiteHandler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) {
	if err := h.handle(writer, request); err != nil {
		infra.HTTPError(writer, err)
	}
}

func (h *RwreiteHandler) handle(writer contracts.ResponseWriter, request *contracts.Request) error {
	if strings.EqualFold(request.Method, http.MethodOptions) {
		h.makeOptionsResponse(writer, request)

		return nil
	}

	originalRequest, err := h.makeOriginalRequest(request)
	if err != nil {
		return fmt.Errorf("failed to create reuest to original source: %w", err)
	}

	originalResponse, err := h.http.Do(originalRequest)
	if err != nil {
		return err
	}

	defer helpers.CloseSafe(originalResponse.Body)

	if err := copyHeaders(originalResponse.Header, writer.Header(), modificationsMap{}); err != nil {
		return err
	}

	infra.WriteCorsHeaders(writer.Header())

	return copyResponseData(writer, originalResponse)
}

func (h *RwreiteHandler) makeOriginalRequest(req *http.Request) (*http.Request, error) {
	req.URL.Path = h.rewrite.To
	if len(h.rewrite.Host) > 0 {
		req.Host = h.rewrite.Host
	}

	originalRequest, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to make requst to original server: %w", err)
	}

	err = copyHeaders(req.Header, originalRequest.Header, modificationsMap{})
	if err != nil {
		return nil, err
	}

	for _, cookie := range req.Cookies() {
		originalRequest.AddCookie(cookie)
	}

	return originalRequest, nil
}
