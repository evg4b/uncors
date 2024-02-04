package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
)

var Spinner = spinner.Spinner{
	Frames: []string{
		"▱▱▱",
		"▰▱▱",
		"▱▰▱",
		"▱▱▰",
	},
	FPS: time.Second / 7, //nolint:gomnd
}
