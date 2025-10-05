package proxy

import (
	"fmt"
	"io"
	"net/http"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/go-http-utils/headers"
)

func (h *Handler) makeUncorsResponse(
	original *http.Response,
	target http.ResponseWriter,
	replacer *urlreplacer.Replacer,
) error {
	copyCookiesToSource(original, replacer, target)

	modifications := modificationsMap{
		headers.Location: func(s string) (string, error) { // nolint: unparam
			return replacer.ReplaceSoft(s), nil
		},
	}

	if err := copyHeaders(original.Header, target.Header(), modifications); err != nil {
		return err
	}

	infra.WriteCorsHeaders(target.Header())

	return copyResponseData(target, original)
}

func copyResponseData(resp http.ResponseWriter, targetResp *http.Response) error {
	resp.WriteHeader(targetResp.StatusCode)

	if _, err := io.Copy(resp, targetResp.Body); err != nil {
		return fmt.Errorf("failed to copy body to response: %w", err)
	}

	return nil
}
