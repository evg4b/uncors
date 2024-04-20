package styles

import (
	"strings"

	"github.com/charmbracelet/log"
)

var (
	DebugLabel   = strings.ToUpper(log.DebugLevel.String())
	InfoLabel    = strings.ToUpper(log.InfoLevel.String())
	WarningLabel = strings.ToUpper(log.WarnLevel.String())
	ErrorLabel   = strings.ToUpper(log.ErrorLevel.String())
	FatalLabel   = strings.ToUpper(log.FatalLevel.String())
)
