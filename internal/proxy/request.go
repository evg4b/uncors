package proxy

import (
	"fmt"
	"net/http"

	"github.com/evg4b/uncors/internal/urlreplacer"
)

func (handler *Handler) makeOriginalRequest(
	req *http.Request,
	replacer *urlreplacer.Replacer,
) (*http.Request, error) {
	url, _ := replacer.Replace(req.URL.String())
	originalReq, err := http.NewRequestWithContext(req.Context(), req.Method, url, req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to make requst to original server: %w", err)
	}

	err = copyHeaders(req.Header, originalReq.Header, modificationsMap{
		"origin":  replacer.Replace,
		"referer": replacer.Replace,
	})

	if err != nil {
		return nil, err
	}

	if err = copyCookiesToTarget(req, replacer, originalReq); err != nil {
		return nil, fmt.Errorf("failed to copy cookies in request: %w", err)
	}

	return originalReq, nil
}
