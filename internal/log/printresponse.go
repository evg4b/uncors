package log

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/pterm/pterm"
)

func printResponse(request *contracts.Request, statusCode int) string {
	prefix := sfmt.Sprintf("%d %s", statusCode, request.Method)
	printer := getPrefixPrinter(statusCode, prefix)

	return printer.Sprint(request.URL.String())
}

func getPrefixPrinter(statusCode int, text string) pterm.PrefixPrinter {
	if statusCode < 100 || statusCode > 599 {
		panic(sfmt.Sprintf("status code %d is not supported", statusCode))
	}

	if 100 <= statusCode && statusCode <= 199 {
		return pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.InfoMessageStyle,
			Prefix: pterm.Prefix{
				Style: &pterm.ThemeDefault.InfoPrefixStyle,
				Text:  text,
			},
		}
	}

	if 200 <= statusCode && statusCode <= 299 {
		return pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.SuccessMessageStyle,
			Prefix: pterm.Prefix{
				Style: &pterm.ThemeDefault.SuccessPrefixStyle,
				Text:  text,
			},
		}
	}

	if 300 <= statusCode && statusCode <= 399 {
		return pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.WarningMessageStyle,
			Prefix: pterm.Prefix{
				Style: &pterm.ThemeDefault.WarningPrefixStyle,
				Text:  text,
			},
		}
	}

	return pterm.PrefixPrinter{
		MessageStyle: &pterm.ThemeDefault.ErrorMessageStyle,
		Prefix: pterm.Prefix{
			Style: &pterm.ThemeDefault.ErrorPrefixStyle,
			Text:  text,
		},
	}
}
