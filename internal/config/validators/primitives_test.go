package validators_test

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runOK(t *testing.T, name string, fn func(*validators.Errors)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		var errs validators.Errors
		fn(&errs)
		assert.False(t, errs.HasAny(), "expected no errors, got: %v", errs)
	})
}

func runErr(t *testing.T, name, expected string, fn func(*validators.Errors)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		var errs validators.Errors
		fn(&errs)
		require.EqualError(t, errs, expected)
	})
}

// ---- ValidateHost --------------------------------------------------------

func TestValidateHost(t *testing.T) {
	const field = "field"

	t.Run("valid", func(t *testing.T) {
		runOK(t, "bare host", func(e *validators.Errors) { validators.ValidateHost(field, hosts.Localhost.Host(), e) })
		runOK(t, "http scheme", func(e *validators.Errors) { validators.ValidateHost(field, hosts.Github.HTTP(), e) })
		runOK(t, "https scheme", func(e *validators.Errors) { validators.ValidateHost(field, hosts.Github.HTTPS(), e) })
		runOK(t, "ip address", func(e *validators.Errors) { validators.ValidateHost(field, hosts.Loopback.Host(), e) })
	})

	t.Run("invalid", func(t *testing.T) {
		runErr(t, "empty", "field must not be empty",
			func(e *validators.Errors) { validators.ValidateHost(field, "", e) })
		runErr(t, "too long", fmt.Sprintf("field must not be longer than 255 characters, but got 256"),
			func(e *validators.Errors) { validators.ValidateHost(field, strings.Repeat("a", 256), e) })
		runErr(t, "with path", "field must not contain a path",
			func(e *validators.Errors) { validators.ValidateHost(field, "example.com/path", e) })
		runErr(t, "with query", "field must not contain a query",
			func(e *validators.Errors) { validators.ValidateHost(field, "example.com?query=1", e) })
		runErr(t, "unsupported scheme", "field scheme must be http or https",
			func(e *validators.Errors) { validators.ValidateHost(field, hosts.Localhost.Scheme("ftp"), e) })
		runErr(t, "invalid host", "field is not a valid host",
			func(e *validators.Errors) { validators.ValidateHost(field, "loca:::lhost", e) })
	})
}

// ---- ValidatePath --------------------------------------------------------

func TestValidatePath(t *testing.T) {
	const field = "field"

	t.Run("valid absolute", func(t *testing.T) {
		runOK(t, "root", func(e *validators.Errors) { validators.ValidatePath(field, "/", false, e) })
		runOK(t, "api path", func(e *validators.Errors) { validators.ValidatePath(field, "/api/info", false, e) })
	})
}

// ---- ValidateFile --------------------------------------------------------

func TestValidateFile(t *testing.T) {
	const field = "test"

	t.Run("valid file", func(t *testing.T) {
		path := "/demo/file.go"
		fs := testutils.FsFromMap(t, map[string]string{path: "package validators"})
		runOK(t, "existing file", func(e *validators.Errors) { validators.ValidateFile(field, path, fs, e) })
	})

	fs := testutils.FsFromMap(t, map[string]string{"file.go": "package validators"})
	testutils.CheckNoError(t, fs.Mkdir("/demo", 0o755))

	runErr(t, "does not exist", "test file_does_not_exist.go does not exist",
		func(e *validators.Errors) { validators.ValidateFile(field, "file_does_not_exist.go", fs, e) })
	runErr(t, "is a directory", "test /demo is a directory",
		func(e *validators.Errors) { validators.ValidateFile(field, "/demo", fs, e) })
}

// ---- ValidateDirectory ---------------------------------------------------

func TestValidateDirectory(t *testing.T) {
	const (
		field = "test"
		dir   = "/demo"
	)

	fs := testutils.FsFromMap(t, map[string]string{"file.go": "package validators"})
	testutils.CheckNoError(t, fs.Mkdir(dir, 0o755))

	runOK(t, "existing directory", func(e *validators.Errors) { validators.ValidateDirectory(field, dir, fs, e) })

	runErr(t, "empty path", "test must not be empty",
		func(e *validators.Errors) { validators.ValidateDirectory(field, "", fs, e) })
	runErr(t, "does not exist", "test directory does not exist",
		func(e *validators.Errors) { validators.ValidateDirectory(field, "does_not_exist", fs, e) })
	runErr(t, "is a file", "test is not a directory",
		func(e *validators.Errors) { validators.ValidateDirectory(field, "file.go", fs, e) })
}

// ---- ValidateStatus ------------------------------------------------------

func TestValidateStatus(t *testing.T) {
	const field = "status"

	for _, code := range []int{100, 200, 300, 400, 404, 500, 503, 599} {
		runOK(t, strconv.Itoa(code), func(e *validators.Errors) { validators.ValidateStatus(field, code, e) })
	}

	for _, code := range []int{-200, 0, 99, 600} {
		runErr(t, strconv.Itoa(code), "status code must be in range 100-599",
			func(e *validators.Errors) { validators.ValidateStatus(field, code, e) })
	}
}

// ---- ValidateDuration ----------------------------------------------------

func TestValidateDuration(t *testing.T) {
	const field = "test-field"

	runOK(t, "positive without allowZero", func(e *validators.Errors) {
		validators.ValidateDuration(field, time.Second, false, e)
	})
	runOK(t, "zero with allowZero", func(e *validators.Errors) {
		validators.ValidateDuration(field, 0, true, e)
	})

	runErr(t, "negative without allowZero", "test-field must be greater than 0",
		func(e *validators.Errors) { validators.ValidateDuration(field, -time.Second, false, e) })
	runErr(t, "zero without allowZero", "test-field must be greater than 0",
		func(e *validators.Errors) { validators.ValidateDuration(field, 0, false, e) })
	runErr(t, "negative with allowZero", "test-field must be greater than or equal to 0",
		func(e *validators.Errors) { validators.ValidateDuration(field, -time.Second, true, e) })
}

// ---- ValidateMethod ------------------------------------------------------

func TestValidateMethod(t *testing.T) {
	const field = "test-field"

	for _, method := range []string{
		http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace,
	} {
		m := method
		runOK(t, fmt.Sprintf("http method %s", m), func(e *validators.Errors) {
			validators.ValidateMethod(field, m, false, e)
		})
	}

	runOK(t, "empty when allowEmpty", func(e *validators.Errors) {
		validators.ValidateMethod(field, "", true, e)
	})

	expected := "test-field must be one of GET, HEAD, POST, PUT, PATCH, DELETE, CONNECT, OPTIONS, TRACE"
	runErr(t, "empty when not allowEmpty", expected,
		func(e *validators.Errors) { validators.ValidateMethod(field, "", false, e) })
	runErr(t, "invalid method", expected,
		func(e *validators.Errors) { validators.ValidateMethod(field, "invalid", false, e) })
}

// ---- ValidatePort --------------------------------------------------------

func TestValidatePort(t *testing.T) {
	const field = "port-field"

	for _, port := range []int{1, 443, 65535} {
		p := port
		runOK(t, fmt.Sprintf("port %d", p), func(e *validators.Errors) { validators.ValidatePort(field, p, e) })
	}

	for _, port := range []int{-5, 0, 70000} {
		p := port
		runErr(t, fmt.Sprintf("port %d", p), "port-field must be between 1 and 65535",
			func(e *validators.Errors) { validators.ValidatePort(field, p, e) })
	}
}

// ---- ValidateGlobPattern -------------------------------------------------

func TestValidateGlobPattern(t *testing.T) {
	runOK(t, "valid glob", func(e *validators.Errors) {
		validators.ValidateGlobPattern("field", "/api/**", e)
	})
	runErr(t, "invalid glob", "field is not a valid glob pattern",
		func(e *validators.Errors) { validators.ValidateGlobPattern("field", "[invalid", e) })
}

// ---- ValidateStringEnum --------------------------------------------------

func TestValidateStringEnum(t *testing.T) {
	options := []string{"option-1", "option-2"}

	runOK(t, "valid option", func(e *validators.Errors) {
		validators.ValidateStringEnum("field", "option-1", options, e)
	})
	runErr(t, "invalid option", "'option-x' is not a valid option",
		func(e *validators.Errors) { validators.ValidateStringEnum("field", "option-x", options, e) })
}
