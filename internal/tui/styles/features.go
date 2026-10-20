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

var DimTextStyle = lipgloss.NewStyle().Foreground(DebugColor)

var handlerStyles = map[events.HandlerKind]lipgloss.Style{
	events.HandlerProxy:   ProxyStyle,
	events.HandlerCache:   CacheStyle,
	events.HandlerMock:    MockStyle,
	events.HandlerScript:  ScriptStyle,
	events.HandlerStatic:  StaticStyle,
	events.HandlerOptions: OptionsStyle,
}

func HandlerLabel(kind events.HandlerKind) string {
	style, ok := handlerStyles[kind]
	if !ok {
		return blockStyle.Render("")
	}

	return style.Render(kind.String())
}
