package uncors

import (
	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/tui/styles"
)

func NewProxyLogger(logger *log.Logger) *log.Logger {
	return tui.CreateLogger(logger, styles.ProxyStyle.Render("PROXY"))
}

func NewOptionsLogger(logger *log.Logger) *log.Logger {
	// TODO: Provide a logger with a specific style
	return tui.CreateLogger(logger, styles.ProxyStyle.Render("OPTNS"))
}

func NewMockLogger(logger *log.Logger) *log.Logger {
	return tui.CreateLogger(logger, styles.MockStyle.Render("MOCK"))
}

func NewStaticLogger(logger *log.Logger) *log.Logger {
	return tui.CreateLogger(logger, styles.StaticStyle.Render("STATIC"))
}

func NewCacheLogger(logger *log.Logger) *log.Logger {
	return tui.CreateLogger(logger, styles.CacheStyle.Render("CACHE"))
}

func NewRewriteLogger(logger *log.Logger) *log.Logger {
	return tui.CreateLogger(logger, styles.RewriteStyle.Render("REWRT"))
}

func NewScriptLogger(logger *log.Logger) *log.Logger {
	return tui.CreateLogger(logger, styles.MockStyle.Render("SCRIPT"))
}
