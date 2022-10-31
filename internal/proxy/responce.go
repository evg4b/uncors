package proxy

import (
	"fmt"
	"net/http"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/internal/urlreplacer"
)

func (handler *Handler) makeUncorsResponse(
	originalResp *http.Response,
	resp http.ResponseWriter,
	replacer *urlreplacer.Replacer,
) error {
	if err := copyCookiesToSource(originalResp, replacer, resp); err != nil {
		return fmt.Errorf("failed to copy cookies in request: %w", err)
	}

	err := copyHeaders(originalResp.Header, resp.Header(), modificationsMap{
		"location": replacer.Replace,
	})
	if err != nil {
		return err
	}

	infrastructure.WriteCorsHeaders(resp.Header())

	if err = copyResponseData(resp, originalResp); err != nil {
		return err
	}

	return nil
}
