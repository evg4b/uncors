package uncors

import (
	"github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/internal/tui/styles"
)

func NewProxyLogger(logger *log.Logger) *log.Logger {
	return log.CreateLogger(logger, styles.ProxyStyle.Render("PROXY"))
}

func NewOptionsLogger(logger *log.Logger) *log.Logger {
	// TODO(design): Create dedicated OptionsStyle in styles package for visual distinction from proxy logs
	return log.CreateLogger(logger, styles.ProxyStyle.Render("OPTNS"))
}

func NewStaticLogger(logger *log.Logger) *log.Logger {
	return log.CreateLogger(logger, styles.StaticStyle.Render("STATIC"))
}

func NewCacheLogger(logger *log.Logger) *log.Logger {
	return log.CreateLogger(logger, styles.CacheStyle.Render("CACHE"))
}

func NewRewriteLogger(logger *log.Logger) *log.Logger {
	return log.CreateLogger(logger, styles.RewriteStyle.Render("REWRT"))
}

func NewScriptLogger(logger *log.Logger) *log.Logger {
	return log.CreateLogger(logger, styles.StaticStyle.Render("SCRIPT"))
}
