package styles

import "charm.land/lipgloss/v2"

var (
	ProxyStyle   = blockStyle.Background(proxyColor)
	MockStyle    = blockStyle.Background(mockColor)
	StaticStyle  = blockStyle.Background(staticColor)
	CacheStyle   = blockStyle.Background(cacheColor)
	RewriteStyle = blockStyle.Background(rewriteColor)
	OptionsStyle = blockStyle.Background(optionsColor)
)

// featureStyles maps a handler's plain name to its badge style. The names come
// from the service, which must not know how a badge looks; the mapping is the
// only place that decision is made.
var featureStyles = map[string]lipgloss.Style{
	"PROXY":   ProxyStyle,
	"MOCK":    MockStyle,
	"STATIC":  StaticStyle,
	"CACHE":   CacheStyle,
	"REWRITE": RewriteStyle,
	"OPTIONS": OptionsStyle,
	"SCRIPT":  RewriteStyle,
}

// Feature renders a handler badge. An unrecognised name is returned unchanged,
// so arbitrary prefixes still work.
func Feature(name string) string {
	if style, ok := featureStyles[name]; ok {
		return style.Render(name)
	}

	return name
}
