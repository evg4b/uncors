package styles_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/stretchr/testify/assert"
)

func TestThemeStatusStyle(t *testing.T) {
	theme := styles.NewTheme(styles.Dark)

	assert.Equal(t, theme.InfoBlock.Render("x"), theme.StatusStyle(101).Render("x"))
	assert.Equal(t, theme.ErrorBlock.Render("x"), theme.StatusStyle(503).Render("x"))

	for _, code := range []int{-150, 0, 99, 600, 999} {
		assert.Equal(t, theme.Cancelled.Render("x"), theme.StatusStyle(code).Render("x"), code)
	}
}

func TestThemeHandlerLabel(t *testing.T) {
	theme := styles.NewTheme(styles.Light)

	assert.Contains(t, theme.HandlerLabel(events.HandlerProxy), "PROXY")
	assert.Contains(t, theme.HandlerLabel(events.HandlerOptions), "OPTNS")
	assert.Equal(t, theme.HandlerLabel(events.HandlerNone), theme.HandlerLabel(events.HandlerKind(200)))
	assert.NotContains(t, theme.HandlerLabel(events.HandlerNone), "NONE")
}
