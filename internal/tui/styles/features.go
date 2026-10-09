package styles

import "charm.land/lipgloss/v2"

// featureStyles maps a handler's plain name to its badge style.
//
// The names come from the service, which must not know how a badge looks, so
// this is the single place that decision is made. SCRIPT shares the rewrite
// colour because both stand for a response the proxy produced itself.
var featureStyles = map[string]lipgloss.Style{
	"PROXY":   blockStyle.Background(proxyColor),
	"MOCK":    blockStyle.Background(mockColor),
	"STATIC":  blockStyle.Background(staticColor),
	"CACHE":   blockStyle.Background(cacheColor),
	"REWRITE": blockStyle.Background(rewriteColor),
	"OPTIONS": blockStyle.Background(optionsColor),
	"SCRIPT":  blockStyle.Background(rewriteColor),
}

// Feature renders a handler badge. An unrecognised name is returned unchanged,
// so arbitrary prefixes still work.
func Feature(name string) string {
	if style, ok := featureStyles[name]; ok {
		return style.Render(name)
	}

	return name
}
