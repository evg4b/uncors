package log

import "github.com/pterm/pterm"

var (
	// infoPrinter returns a PrefixPrinter, which can be used to print text with an "infoPrinter" Prefix.
	infoPrinter = pterm.PrefixPrinter{
		MessageStyle: &pterm.ThemeDefault.InfoMessageStyle,
		Prefix: pterm.Prefix{
			Style: &pterm.ThemeDefault.InfoPrefixStyle,
			Text:  "   INFO",
		},
	}

	// warningPrinter returns a PrefixPrinter, which can be used to print text with a "warningPrinter" Prefix.
	warningPrinter = pterm.PrefixPrinter{
		MessageStyle: &pterm.ThemeDefault.WarningMessageStyle,
		Prefix: pterm.Prefix{
			Style: &pterm.ThemeDefault.WarningPrefixStyle,
			Text:  "WARNING",
		},
	}

	// errorPrinter returns a PrefixPrinter, which can be used to print text with an "errorPrinter" Prefix.
	errorPrinter = pterm.PrefixPrinter{
		MessageStyle: &pterm.ThemeDefault.ErrorMessageStyle,
		Prefix: pterm.Prefix{
			Style: &pterm.ThemeDefault.ErrorPrefixStyle,
			Text:  "  ERROR",
		},
	}

	// debugPrinter Prints debugPrinter messages. By default, it will only print if PrintDebugMessages is true.
	// You can change PrintDebugMessages with EnableDebugMessages and DisableDebugMessages,
	// or by setting the variable itself.
	debugPrinter = pterm.PrefixPrinter{
		MessageStyle: &pterm.ThemeDefault.DebugMessageStyle,
		Prefix: pterm.Prefix{
			Text:  "  DEBUG",
			Style: &pterm.ThemeDefault.DebugPrefixStyle,
		},
		Debugger: true,
	}
)
