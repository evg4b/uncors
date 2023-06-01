package proxy

import (
	"fmt"
	"net/http"

	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/go-http-utils/headers"
)

func (m *Handler) makeOriginalRequest(
	req *http.Request,
	replacer *urlreplacer.Replacer,
) (*http.Request, error) {
	url, _ := replacer.Replace(req.URL.String())
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

	if err = copyCookiesToTarget(req, replacer, originalRequest); err != nil {
		return nil, fmt.Errorf("failed to copy cookies in request: %w", err)
	}

	return originalRequest, nil
}
