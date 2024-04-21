package uncors

import (
	// "github.com/evg4b/uncors/internal/log".
	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/tui/styles"
)

func NewProxyLogger(logger *log.Logger) *log.Logger {
	return styles.CreateLogger(logger, styles.ProxyStyle.Render("PROXY"))
}

func NewMockLogger(logger *log.Logger) *log.Logger {
	return styles.CreateLogger(logger, styles.MockStyle.Render("MOCK"))
}

func NewStaticLogger(logger *log.Logger) *log.Logger {
	return styles.CreateLogger(logger, styles.StaticStyle.Render("STATIC"))
}

func NewCacheLogger(logger *log.Logger) *log.Logger {
	return styles.CreateLogger(logger, styles.CacheStyle.Render("CACHE"))
}
