package ui

import (
	"github.com/evg4b/uncors/internal/log"
	"github.com/pterm/pterm"
)

var ProxyLogger = log.NewLogger(" PROXY ", log.WithStyle(&pterm.Style{
	pterm.FgBlack,
	pterm.BgLightBlue,
}))

var MockLogger = log.NewLogger(" MOCK  ", log.WithStyle(&pterm.Style{
	pterm.FgBlack,
	pterm.BgLightMagenta,
}))

var StaticLogger = log.NewLogger("STATIC ", log.WithStyle(&pterm.Style{
	pterm.FgBlack,
	pterm.BgLightYellow,
}))
