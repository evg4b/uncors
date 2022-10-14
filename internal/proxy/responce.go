package proxy

import (
	"fmt"
	"net/http"

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

	header := resp.Header()
	err := copyHeaders(originalResp.Header, header, modificationsMap{
		"location": replacer.Replace,
	})

	if err != nil {
		return err
	}

	if err = copyResponseData(header, resp, originalResp); err != nil {
		return err
	}

	return nil
}
