package uncors

import (
	"github.com/charmbracelet/log"
)

// var (
//	ProxyLogger  = log.NewLogger(" PROXY  ", style(pterm.FgBlack, pterm.BgLightBlue))
//	MockLogger   = log.NewLogger(" MOCK   ", style(pterm.FgBlack, pterm.BgLightMagenta))
//	StaticLogger = log.NewLogger(" STATIC ", style(pterm.FgBlack, pterm.BgLightWhite))
//	CacheLogger  = log.NewLogger(" CACHE  ", style(pterm.FgBlack, pterm.BgLightYellow))
//)

var (
	ProxyLogger  = log.Default()
	MockLogger   = log.Default()
	StaticLogger = log.Default()
	CacheLogger  = log.Default()
)
