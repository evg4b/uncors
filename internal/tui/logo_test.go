package tui_test

import (
	"testing"

	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/evg4b/uncors/internal/tui"
)

func TestLogo(t *testing.T) {
	t.Run("Logo", testutils.WithTrueColor(func(t *testing.T) {
		version := "0.1.0"
		logo := tui.Logo(version)

		snaps.MatchSnapshot(t, logo)
	}))
}
