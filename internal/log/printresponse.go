package log

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/pterm/pterm"
)

func printResponse(request *contracts.Request, statusCode int) string {
	prefix := helpers.Sprintf("%d %s", statusCode, request.Method)
	printer := getPrefixPrinter(statusCode, prefix)

	return printer.Sprint(request.URL.String())
}

func getPrefixPrinter(statusCode int, text string) pterm.PrefixPrinter {
	if helpers.Is1xxCode(statusCode) {
		return pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.InfoMessageStyle,
			Prefix: pterm.Prefix{
				Style: &pterm.ThemeDefault.InfoPrefixStyle,
				Text:  text,
			},
		}
	}

	if helpers.Is2xxCode(statusCode) {
		return pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.SuccessMessageStyle,
			Prefix: pterm.Prefix{
				Style: &pterm.ThemeDefault.SuccessPrefixStyle,
				Text:  text,
			},
		}
	}

	if helpers.Is3xxCode(statusCode) {
		return pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.WarningMessageStyle,
			Prefix: pterm.Prefix{
				Style: &pterm.ThemeDefault.WarningPrefixStyle,
				Text:  text,
			},
		}
	}

	if helpers.Is4xxCode(statusCode) || helpers.Is5xxCode(statusCode) {
		return pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.ErrorMessageStyle,
			Prefix: pterm.Prefix{
				Style: &pterm.ThemeDefault.ErrorPrefixStyle,
				Text:  text,
			},
		}
	}

	panic(helpers.Sprintf("status code %d is not supported", statusCode))
}
