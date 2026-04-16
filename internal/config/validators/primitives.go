package validators

import (
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/evg4b/uncors/internal/urlparser"
	"github.com/spf13/afero"
)

const maxHostLength = 255

func ValidateHost(field, value string, errs *Errors) {
	if value == "" {
		errs.add(fmt.Sprintf("%s must not be empty", field))

		return
	}

	if len(value) > maxHostLength {
		errs.add(fmt.Sprintf("%s must not be longer than 255 characters, but got %d", field, len(value)))

		return
	}

	uri, err := urlparser.Parse(value)
	if err != nil {
		errs.add(fmt.Sprintf("%s is not a valid host", field))

		return
	}

	if uri.Path != "" {
		errs.add(fmt.Sprintf("%s must not contain a path", field))
	}

	if uri.RawQuery != "" {
		errs.add(fmt.Sprintf("%s must not contain a query", field))
	}

	if uri.Scheme != "http" && uri.Scheme != "https" && uri.Scheme != "" {
		errs.add(fmt.Sprintf("%s scheme must be http or https", field))
	}
}

func ValidatePath(field, value string, relative bool, errs *Errors) {
	if value == "" {
		errs.add(fmt.Sprintf("%s must not be empty", field))

		return
	}

	if !relative && !strings.HasPrefix(value, "/") {
		errs.add(fmt.Sprintf("%s must be absolute and start with /", field))

		return
	}

	uri, err := urlparser.Parse("//localhost/" + strings.TrimPrefix(value, "/"))
	if err != nil {
		errs.add(fmt.Sprintf("%s is not a valid path", field))

		return
	}

	if uri.RawQuery != "" {
		errs.add(fmt.Sprintf("%s must not contain a query", field))
	}
}

func ValidateFile(field, value string, fs afero.Fs, errs *Errors) {
	stat, err := fs.Stat(value)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			errs.add(fmt.Sprintf("%s %s does not exist", field, value))
		case os.IsPermission(err):
			errs.add(fmt.Sprintf("%s %s is not accessible", field, value))
		default:
			errs.add(fmt.Sprintf("%s %s is not a file", field, value))
		}

		return
	}

	if stat.IsDir() {
		errs.add(fmt.Sprintf("%s %s is a directory", field, value))
	}
}

func ValidateDirectory(field, value string, fs afero.Fs, errs *Errors) {
	if value == "" {
		errs.add(fmt.Sprintf("%s must not be empty", field))

		return
	}

	stat, err := fs.Stat(value)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			errs.add(fmt.Sprintf("%s directory does not exist", field))
		case os.IsPermission(err):
			errs.add(fmt.Sprintf("%s directory is not accessible", field))
		default:
			errs.add(fmt.Sprintf("%s is not a directory", field))
		}

		return
	}

	if !stat.IsDir() {
		errs.add(fmt.Sprintf("%s is not a directory", field))
	}
}

func ValidateStatus(field string, value int, errs *Errors) {
	if value < 100 || value > 599 {
		errs.add(fmt.Sprintf("%s code must be in range 100-599", field))
	}
}

func ValidateDuration(field string, value time.Duration, allowZero bool, errs *Errors) {
	if allowZero {
		if value < 0 {
			errs.add(fmt.Sprintf("%s must be greater than or equal to 0", field))
		}
	} else {
		if value <= 0 {
			errs.add(fmt.Sprintf("%s must be greater than 0", field))
		}
	}
}

var allowedMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodOptions,
	http.MethodTrace,
}

func ValidateMethod(field, value string, allowEmpty bool, errs *Errors) {
	if allowEmpty && value == "" {
		return
	}

	if !slices.Contains(allowedMethods, value) {
		errs.add(fmt.Sprintf("%s must be one of %s", field, strings.Join(allowedMethods, ", ")))
	}
}

func ValidatePort(field string, value int, errs *Errors) {
	if value < 1 || value > 65535 {
		errs.add(fmt.Sprintf("%s must be between 1 and 65535", field))
	}
}

func ValidateGlobPattern(field, value string, errs *Errors) {
	if !doublestar.ValidatePathPattern(value) {
		errs.add(fmt.Sprintf("%s is not a valid glob pattern", field))
	}
}

func ValidateStringEnum(_ string, value string, options []string, errs *Errors) {
	if !slices.Contains(options, value) {
		errs.add(fmt.Sprintf("'%s' is not a valid option", value))
	}
}
