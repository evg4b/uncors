package uncors

import (
	"github.com/charmbracelet/log"
)

var (
	ProxyLogger  = log.Default().WithPrefix("proxy")
	MockLogger   = log.Default().WithPrefix("mock")
	StaticLogger = log.Default().WithPrefix("static")
	CacheLogger  = log.Default().WithPrefix("cache")
)
