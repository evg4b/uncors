package styles

import (
	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/events"
)

var (
	ProxyStyle   = blockStyle.Background(proxyColor)
	MockStyle    = blockStyle.Background(mockColor)
	StaticStyle  = blockStyle.Background(staticColor)
	CacheStyle   = blockStyle.Background(cacheColor)
	ScriptStyle  = blockStyle.Background(scriptColor)
	OptionsStyle = blockStyle.Background(optionsColor)
)

// DimTextStyle renders secondary details such as a rewritten path or a
// handler error.
var DimTextStyle = lipgloss.NewStyle().Foreground(DebugColor)

var handlerStyles = map[events.HandlerKind]lipgloss.Style{
	events.HandlerProxy:   ProxyStyle,
	events.HandlerCache:   CacheStyle,
	events.HandlerMock:    MockStyle,
	events.HandlerScript:  ScriptStyle,
	events.HandlerStatic:  StaticStyle,
	events.HandlerOptions: OptionsStyle,
}

// HandlerLabel renders the block that names the component that handled a
// request, or an empty string when no component did.
func HandlerLabel(kind events.HandlerKind) string {
	style, ok := handlerStyles[kind]
	if !ok {
		return ""
	}

	return style.Render(kind.String())
}
