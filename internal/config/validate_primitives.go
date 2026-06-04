package config

import (
	"errors"
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

func ValidateHost(field, value string) error {
	if value == "" {
		return &ValidationError{fmt.Sprintf("%s must not be empty", field)}
	}

	if len(value) > maxHostLength {
		return &ValidationError{fmt.Sprintf("%s must not be longer than 255 characters, but got %d", field, len(value))}
	}

	uri, err := urlparser.Parse(value)
	if err != nil {
		return &ValidationError{fmt.Sprintf("%s is not a valid host", field)}
	}

	var errs []error

	if uri.Path != "" {
		errs = append(errs, &ValidationError{fmt.Sprintf("%s must not contain a path", field)})
	}

	if uri.RawQuery != "" {
		errs = append(errs, &ValidationError{fmt.Sprintf("%s must not contain a query", field)})
	}

	if uri.Scheme != "http" && uri.Scheme != httpsScheme && uri.Scheme != "" {
		errs = append(errs, &ValidationError{fmt.Sprintf("%s scheme must be http or https", field)})
	}

	return errors.Join(errs...)
}

func ValidatePath(field, value string, relative bool) error {
	if value == "" {
		return &ValidationError{fmt.Sprintf("%s must not be empty", field)}
	}

	if !relative && !strings.HasPrefix(value, "/") {
		return &ValidationError{fmt.Sprintf("%s must be absolute and start with /", field)}
	}

	uri, err := urlparser.Parse("//localhost/" + strings.TrimPrefix(value, "/"))
	if err != nil {
		return &ValidationError{fmt.Sprintf("%s is not a valid path", field)}
	}

	if uri.RawQuery != "" {
		return &ValidationError{fmt.Sprintf("%s must not contain a query", field)}
	}

	return nil
}

func ValidateFile(field, value string, fs afero.Fs) error {
	stat, err := fs.Stat(value)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			return &ValidationError{fmt.Sprintf("%s %s does not exist", field, value)}
		case os.IsPermission(err):
			return &ValidationError{fmt.Sprintf("%s %s is not accessible", field, value)}
		default:
			return &ValidationError{fmt.Sprintf("%s %s is not a file", field, value)}
		}
	}

	if stat.IsDir() {
		return &ValidationError{fmt.Sprintf("%s %s is a directory", field, value)}
	}

	return nil
}

func ValidateDirectory(field, value string, fs afero.Fs) error {
	if value == "" {
		return &ValidationError{fmt.Sprintf("%s must not be empty", field)}
	}

	stat, err := fs.Stat(value)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			return &ValidationError{fmt.Sprintf("%s directory does not exist", field)}
		case os.IsPermission(err):
			return &ValidationError{fmt.Sprintf("%s directory is not accessible", field)}
		default:
			return &ValidationError{fmt.Sprintf("%s is not a directory", field)}
		}
	}

	if !stat.IsDir() {
		return &ValidationError{fmt.Sprintf("%s is not a directory", field)}
	}

	return nil
}

func ValidateStatus(field string, value int) error {
	if value < 100 || value > 599 {
		return &ValidationError{fmt.Sprintf("%s code must be in range 100-599", field)}
	}

	return nil
}

func ValidateDuration(field string, value time.Duration, allowZero bool) error {
	if allowZero {
		if value < 0 {
			return &ValidationError{fmt.Sprintf("%s must be greater than or equal to 0", field)}
		}
	} else {
		if value <= 0 {
			return &ValidationError{fmt.Sprintf("%s must be greater than 0", field)}
		}
	}

	return nil
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

func ValidateMethod(field, value string, allowEmpty bool) error {
	if allowEmpty && value == "" {
		return nil
	}

	if !slices.Contains(allowedMethods, value) {
		return &ValidationError{fmt.Sprintf("%s must be one of %s", field, strings.Join(allowedMethods, ", "))}
	}

	return nil
}

func ValidatePort(field string, value int) error {
	if value < 1 || value > 65535 {
		return &ValidationError{fmt.Sprintf("%s must be between 1 and 65535", field)}
	}

	return nil
}

func ValidateGlobPattern(field, value string) error {
	if !doublestar.ValidatePathPattern(value) {
		return &ValidationError{fmt.Sprintf("%s is not a valid glob pattern", field)}
	}

	return nil
}

func ValidateStringEnum(_ string, value string, options []string) error {
	if !slices.Contains(options, value) {
		return &ValidationError{fmt.Sprintf("'%s' is not a valid option", value)}
	}

	return nil
}
