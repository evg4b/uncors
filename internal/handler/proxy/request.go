package proxy

import (
	"fmt"
	"net/http"

	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/go-http-utils/headers"
)

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
		return nil, fmt.Errorf("failed to make requst to original server: %w", err)
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
