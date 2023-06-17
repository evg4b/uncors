package ui

import (
	"github.com/evg4b/uncors/internal/log"
	"github.com/pterm/pterm"
)

func style(fg pterm.Color, bg pterm.Color) log.LoggerOption {
	return log.WithStyle(&pterm.Style{fg, bg})
}

var (
	ProxyLogger  = log.NewLogger(" PROXY ", style(pterm.FgBlack, pterm.BgLightBlue))
	MockLogger   = log.NewLogger(" MOCK  ", style(pterm.FgBlack, pterm.BgLightMagenta))
	StaticLogger = log.NewLogger("STATIC ", style(pterm.FgBlack, pterm.BgLightWhite))
)
