package infra

import (
	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/muesli/termenv"
)

func ConfigureLogger() {
	log.SetReportTimestamp(false)
	log.SetReportCaller(false)
	log.SetStyles(&styles.DefaultStyles)
	log.SetColorProfile(termenv.ColorProfile())
}
