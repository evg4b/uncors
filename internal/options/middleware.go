package options

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/internal/responceprinter"
	"github.com/pterm/pterm"
)

type OptionsMiddleware struct{}

func NewOptionsMiddlewareMiddleware(options ...optionsMiddlewareOption) *OptionsMiddleware {
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

	return func(w http.ResponseWriter, r *http.Request) error {
		if r.Method != "OPTIONS" {
			return next(w, r)
		}

		header := w.Header()
		for key, values := range r.Header {
			lowerKey := strings.ToLower(key)
			if strings.Contains(lowerKey, "access-control-request") {
				for _, value := range values {
					transformedKey := strings.Replace(lowerKey, "request", "allow", 1)
					header.Add(transformedKey, value)
				}
			}
		}

		optionsWriter.Printfln(responceprinter.PrintResponce(&http.Response{
			StatusCode: http.StatusOK,
			Request:    r,
		}))

		return nil
	}
}
