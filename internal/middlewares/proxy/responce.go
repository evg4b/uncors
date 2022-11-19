package proxy

import (
	"fmt"
	"io"
	"net/http"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/go-http-utils/headers"
)

func (m *Middleware) makeUncorsResponse(
	original *http.Response,
	target http.ResponseWriter,
	replacer *urlreplacer.Replacer,
) error {
	if err := copyCookiesToSource(original, replacer, target); err != nil {
		return fmt.Errorf("failed to copy cookies in request: %w", err)
	}

	err := copyHeaders(original.Header, target.Header(), modificationsMap{
		headers.Location: replacer.Replace,
	})
	if err != nil {
		return err
	}

	infrastructure.WriteCorsHeaders(target.Header())

	if err = copyResponseData(target, original); err != nil {
		return err
	}

	return nil
}

func copyResponseData(resp http.ResponseWriter, targetResp *http.Response) error {
	resp.WriteHeader(targetResp.StatusCode)

	if _, err := io.Copy(resp, targetResp.Body); err != nil {
		return fmt.Errorf("failed to copy body to response: %w", err)
	}

	return nil
}
