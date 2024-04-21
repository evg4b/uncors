package uncors

import (
	// "github.com/evg4b/uncors/internal/log".
	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/tui/styles"
)

func NewProxyLogger(logger *log.Logger) *log.Logger {
	return logger.WithPrefix(styles.ProxyStyle.Render("PROXY"))
}

func NewMockLogger(logger *log.Logger) *log.Logger {
	return logger.WithPrefix(styles.MockStyle.Render("MOCK"))
}

func NewStaticLogger(logger *log.Logger) *log.Logger {
	return logger.WithPrefix(styles.StaticStyle.Render("STATIC"))
}

func NewCacheLogger(logger *log.Logger) *log.Logger {
	return logger.WithPrefix(styles.CacheStyle.Render("CACHE"))
}
