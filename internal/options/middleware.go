package options

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/internal/responseprinter"
	"github.com/pterm/pterm"
)

type OptionsMiddleware struct{} // nolint: revive

func NewOptionsMiddleware(options ...optionsMiddlewareOption) *OptionsMiddleware {
	middleware := &OptionsMiddleware{}

	for _, option := range options {
		option(middleware)
	}

	return middleware
}

func (pm *OptionsMiddleware) Wrap(next infrastructure.HandlerFunc) infrastructure.HandlerFunc {
	optionsWriter := pterm.PrefixPrinter{
		MessageStyle: &pterm.ThemeDefault.InfoMessageStyle,
		Prefix: pterm.Prefix{
			Style: &pterm.Style{pterm.FgBlack, pterm.BgLightGreen},
			Text:  "OPTIONS",
		},
	}

	return func(w http.ResponseWriter, req *http.Request) error {
		if req.Method != "OPTIONS" {
			return next(w, req)
		}

		header := w.Header()
		for key, values := range req.Header {
			lowerKey := strings.ToLower(key)
			if strings.Contains(lowerKey, "access-control-request") {
				for _, value := range values {
					transformedKey := strings.Replace(lowerKey, "request", "allow", 1)
					header.Add(transformedKey, value)
				}
			}
		}

		optionsWriter.Printfln(responseprinter.Printresponse(&http.Response{
			StatusCode: http.StatusOK,
			Request:    req,
		}))

		return nil
	}
}
