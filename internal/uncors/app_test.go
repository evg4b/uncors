package uncors_test

import (
	"testing"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestUncorsApp(t *testing.T) {
	app := uncors.CreateApp(afero.NewMemMapFs(), log.Default(), "x.x.x")

	err := app.Start(t.Context(), &config.UncorsConfig{})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, app.Close())
	}()
}
