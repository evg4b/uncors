package validators

import (
	"fmt"
	"path/filepath"

	"github.com/evg4b/uncors/internal/config"
)

func ValidateHAR(field string, value config.HARConfig, errs *Errors) {
	if !value.Enabled() {
		return
	}

	file := value.File

	if filepath.Ext(file) == "" {
		errs.add(fmt.Sprintf("%s: HAR file path %q must have a file extension (e.g. .har)", field, file))
	}
}
