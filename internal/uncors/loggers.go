package uncors

import (
	"os"
	// "github.com/evg4b/uncors/internal/log".
	"github.com/charmbracelet/log"
)

var ProxyLogger = log.NewWithOptions(os.Stdout, log.Options{
	Prefix: " PROXY  ",
})

var MockLogger = log.NewWithOptions(os.Stdout, log.Options{
	Prefix: " MOCK   ",
})

var StaticLogger = log.NewWithOptions(os.Stdout, log.Options{
	Prefix: " STATIC ",
})

var CacheLogger = log.NewWithOptions(os.Stdout, log.Options{
	Prefix: " CACHE  ",
})

//
// var (
//	ProxyLogger  = log.NewLogger(" PROXY  ", style(pterm.FgBlack, pterm.BgLightBlue))
//	MockLogger   = log.NewLogger(" MOCK   ", style(pterm.FgBlack, pterm.BgLightMagenta))
//	StaticLogger = log.NewLogger(" STATIC ", style(pterm.FgBlack, pterm.BgLightWhite))
//	CacheLogger  = log.NewLogger(" CACHE  ", style(pterm.FgBlack, pterm.BgLightYellow))
//)
