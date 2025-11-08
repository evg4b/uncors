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
	req *http.Request,
) error {
	copyCookiesToSource(original, replacer, target)

	modifications := modificationsMap{
		headers.Location: func(s string) (string, error) { // nolint: unparam
			return replacer.ReplaceSoft(s), nil
		},
	}

	err := copyHeaders(original.Header, target.Header(), modifications)
	if err != nil {
		return err
	}

	origin := req.Header.Get(headers.Origin)
	infra.WriteCorsHeaders(target.Header(), origin)

	return copyResponseData(target, original)
}

func copyResponseData(resp http.ResponseWriter, targetResp *http.Response) error {
	resp.WriteHeader(targetResp.StatusCode)

	_, err := io.Copy(resp, targetResp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy body to response: %w", err)
	}

	return nil
}
