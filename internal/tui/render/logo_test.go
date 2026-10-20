package render_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/tui/render"
	"github.com/evg4b/uncors/testing/testutils"
)

func TestLogo(t *testing.T) {
	t.Run("Logo", func(t *testing.T) {
		version := "0.1.0"
		logo := render.Logo(version)

		testutils.MatchSnapshot(t, logo)
	})
}
