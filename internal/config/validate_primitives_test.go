package config_test

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runOK(t *testing.T, name string, fn func() error) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		assert.NoError(t, fn())
	})
}

func runErr(t *testing.T, name, expected string, fn func() error) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		require.EqualError(t, fn(), expected)
	})
}

// ---- ValidateHost --------------------------------------------------------

func TestValidateHost(t *testing.T) {
	const field = "field"

	t.Run("valid", func(t *testing.T) {
		runOK(t, "bare host", func() error { return config.ValidateHost(field, hosts.Localhost.Host().String()) })
		runOK(t, "http scheme", func() error { return config.ValidateHost(field, hosts.Github.HTTP().String()) })
		runOK(t, "https scheme", func() error { return config.ValidateHost(field, hosts.Github.HTTPS().String()) })
		runOK(t, "ip address", func() error { return config.ValidateHost(field, hosts.Loopback.Host().String()) })
	})

	t.Run("invalid", func(t *testing.T) {
		runErr(t, "empty", "field must not be empty",
			func() error { return config.ValidateHost(field, "") })
		runErr(t, "too long", "field must not be longer than 255 characters, but got 256",
			func() error { return config.ValidateHost(field, strings.Repeat("a", 256)) })
		runErr(t, "with path", "field must not contain a path",
			func() error { return config.ValidateHost(field, "example.com/path") })
		runErr(t, "with query", "field must not contain a query",
			func() error { return config.ValidateHost(field, "example.com?query=1") })
		runErr(t, "unsupported scheme", "field scheme must be http or https",
			func() error { return config.ValidateHost(field, hosts.Localhost.Scheme("ftp").String()) })
		runErr(t, "invalid host", "field is not a valid host",
			func() error { return config.ValidateHost(field, "loca:::lhost") })
	})
}

// ---- ValidatePath --------------------------------------------------------

func TestValidatePath(t *testing.T) {
	const field = "field"

	t.Run("valid absolute", func(t *testing.T) {
		runOK(t, "root", func() error { return config.ValidatePath(field, "/", false) })
		runOK(t, "api path", func() error { return config.ValidatePath(field, "/api/info", false) })
	})
}

// ---- ValidateFile --------------------------------------------------------

func TestValidateFile(t *testing.T) {
	const field = "test"

	t.Run("valid file", func(t *testing.T) {
		path := "/demo/file.go"
		fs := testutils.FsFromMap(t, map[string]string{path: "package validators"})
		runOK(t, "existing file", func() error { return config.ValidateFile(field, path, fs) })
	})

	fs := testutils.FsFromMap(t, map[string]string{"file.go": "package validators"})
	testutils.CheckNoError(t, fs.Mkdir("/demo", 0o755))

	runErr(t, "does not exist", "test file_does_not_exist.go does not exist",
		func() error { return config.ValidateFile(field, "file_does_not_exist.go", fs) })
	runErr(t, "is a directory", "test /demo is a directory",
		func() error { return config.ValidateFile(field, "/demo", fs) })
}

// ---- ValidateDirectory ---------------------------------------------------

func TestValidateDirectory(t *testing.T) {
	const (
		field = "test"
		dir   = "/demo"
	)

	fs := testutils.FsFromMap(t, map[string]string{"file.go": "package validators"})
	testutils.CheckNoError(t, fs.Mkdir(dir, 0o755))

	runOK(t, "existing directory", func() error { return config.ValidateDirectory(field, dir, fs) })

	runErr(t, "empty path", "test must not be empty",
		func() error { return config.ValidateDirectory(field, "", fs) })
	runErr(t, "does not exist", "test directory does not exist",
		func() error { return config.ValidateDirectory(field, "does_not_exist", fs) })
	runErr(t, "is a file", "test is not a directory",
		func() error { return config.ValidateDirectory(field, "file.go", fs) })
}

// ---- ValidateStatus ------------------------------------------------------

func TestValidateStatus(t *testing.T) {
	const field = "status"

	for _, code := range []int{100, 200, 300, 400, 404, 500, 503, 599} {
		runOK(t, strconv.Itoa(code), func() error { return config.ValidateStatus(field, code) })
	}

	for _, code := range []int{-200, 0, 99, 600} {
		runErr(t, strconv.Itoa(code), "status code must be in range 100-599",
			func() error { return config.ValidateStatus(field, code) })
	}
}

// ---- ValidateDuration ----------------------------------------------------

func TestValidateDuration(t *testing.T) {
	const field = "test-field"

	runOK(t, "positive without allowZero", func() error {
		return config.ValidateDuration(field, time.Second, false)
	})
	runOK(t, "zero with allowZero", func() error {
		return config.ValidateDuration(field, 0, true)
	})

	runErr(t, "negative without allowZero", "test-field must be greater than 0",
		func() error { return config.ValidateDuration(field, -time.Second, false) })
	runErr(t, "zero without allowZero", "test-field must be greater than 0",
		func() error { return config.ValidateDuration(field, 0, false) })
	runErr(t, "negative with allowZero", "test-field must be greater than or equal to 0",
		func() error { return config.ValidateDuration(field, -time.Second, true) })
}

// ---- ValidateMethod ------------------------------------------------------

func TestValidateMethod(t *testing.T) {
	const field = "test-field"

	for _, method := range []string{
		http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace,
	} {
		m := method
		runOK(t, fmt.Sprintf("http method %s", m), func() error {
			return config.ValidateMethod(field, m, false)
		})
	}

	runOK(t, "empty when allowEmpty", func() error {
		return config.ValidateMethod(field, "", true)
	})

	expected := "test-field must be one of GET, HEAD, POST, PUT, PATCH, DELETE, CONNECT, OPTIONS, TRACE"
	runErr(t, "empty when not allowEmpty", expected,
		func() error { return config.ValidateMethod(field, "", false) })
	runErr(t, "invalid method", expected,
		func() error { return config.ValidateMethod(field, "invalid", false) })
}

// ---- ValidatePort --------------------------------------------------------

func TestValidatePort(t *testing.T) {
	const field = "port-field"

	for _, port := range []int{1, 443, 65535} {
		p := port
		runOK(t, fmt.Sprintf("port %d", p), func() error { return config.ValidatePort(field, p) })
	}

	for _, port := range []int{-5, 0, 70000} {
		p := port
		runErr(t, fmt.Sprintf("port %d", p), "port-field must be between 1 and 65535",
			func() error { return config.ValidatePort(field, p) })
	}
}

// ---- ValidateGlobPattern -------------------------------------------------

func TestValidateGlobPattern(t *testing.T) {
	runOK(t, "valid glob", func() error {
		return config.ValidateGlobPattern("field", "/api/**")
	})
	runErr(t, "invalid glob", "field is not a valid glob pattern",
		func() error { return config.ValidateGlobPattern("field", "[invalid") })
}

// ---- ValidateStringEnum --------------------------------------------------

func TestValidateStringEnum(t *testing.T) {
	options := []string{"option-1", "option-2"}

	runOK(t, "valid option", func() error {
		return config.ValidateStringEnum("field", "option-1", options)
	})
	runErr(t, "invalid option", "'option-x' is not a valid option",
		func() error { return config.ValidateStringEnum("field", "option-x", options) })
}
