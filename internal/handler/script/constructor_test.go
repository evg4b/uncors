package script_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/handler/script"
	"github.com/stretchr/testify/assert"
)

func TestNewHandler_RequiresScript(t *testing.T) {
	assert.PanicsWithValue(
		t,
		"ScriptHandler: Script is not configured",
		func() { script.NewHandler() },
	)
}
