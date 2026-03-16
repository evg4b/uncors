package validators

import (
	"fmt"
	"path"
	"strings"

	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/evg4b/uncors/internal/urlparser"

	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/afero"
)

func ValidateProxy(field, value string, errs *Errors) {
	if value == "" {
		return
	}

	_, err := urlparser.Parse(value)
	if err != nil {
		errs.add(fmt.Sprintf("%s is not a valid URL", field))
	}
}

func ValidateOptionsHandling(field string, value config.OptionsHandling, errs *Errors) {
	if value.Code != 0 {
		ValidateStatus(joinPath(field, "code"), value.Code, errs)
	}
}

func ValidateRequestMatcher(field string, value config.RequestMatcher, errs *Errors) {
	ValidatePath(joinPath(field, "path"), value.Path, false, errs)
	ValidateMethod(joinPath(field, "method"), value.Method, true, errs)
}

func ValidateRewritingOption(field string, value config.RewritingOption, errs *Errors) {
	ValidatePath(joinPath(field, "from"), value.From, true, errs)
	ValidatePath(joinPath(field, "to"), value.To, true, errs)

	if value.Host != "" {
		ValidateHost(joinPath(field, "host"), value.Host, errs)
	}
}

func ValidateCacheGlob(field, value string, errs *Errors) {
	ValidateGlobPattern(field, value, errs)
}

func ValidateResponse(field string, value config.Response, fs afero.Fs, errs *Errors) {
	ValidateStatus(joinPath(field, "code"), value.Code, errs)
	ValidateDuration(joinPath(field, "delay"), value.Delay, true, errs)

	switch {
	case value.Raw == "" && value.File == "":
		errs.add(fmt.Sprintf(
			"%s or %s must be set",
			joinPath(field, "raw"),
			joinPath(field, "file"),
		))
	case value.Raw != "" && value.File != "":
		errs.add(fmt.Sprintf(
			"only one of %s or %s must be set",
			joinPath(field, "raw"),
			joinPath(field, "file"),
		))
	case value.File != "":
		ValidateFile(joinPath(field, "file"), value.File, fs, errs)
	}
}

func ValidateMock(field string, value config.Mock, fs afero.Fs, errs *Errors) {
	ValidateRequestMatcher(field, value.Matcher, errs)
	ValidateResponse(joinPath(field, "response"), value.Response, fs, errs)
}

func ValidateStatic(field string, value config.StaticDirectory, fs afero.Fs, errs *Errors) {
	ValidatePath(joinPath(field, "path"), value.Path, false, errs)
	ValidateDirectory(joinPath(field, "directory"), value.Dir, fs, errs)

	if value.Index != "" {
		ValidateFile(joinPath(field, "index"), path.Join(value.Dir, value.Index), fs, errs)
	}
}

func ValidateScript(field string, value config.Script, fs afero.Fs, errs *Errors) {
	ValidateRequestMatcher(field, value.Matcher, errs)

	switch {
	case value.Script == "" && value.File == "":
		scriptField := joinPath(field, "script")
		fileField := joinPath(field, "file")

		errs.add(fmt.Sprintf("%s: either 'script' or 'file' must be provided", scriptField))
		errs.add(fmt.Sprintf("%s: either 'script' or 'file' must be provided", fileField))
	case value.Script != "" && value.File != "":
		scriptField := joinPath(field, "script")
		fileField := joinPath(field, "file")

		errs.add(fmt.Sprintf("%s: only one of 'script' or 'file' can be provided", scriptField))
		errs.add(fmt.Sprintf("%s: only one of 'script' or 'file' can be provided", fileField))
	case value.File != "":
		ValidateFile(joinPath(field, "file"), value.File, fs, errs)
	}
}

func ValidateCacheConfig(field string, value config.CacheConfig, errs *Errors) {
	ValidateDuration(joinPath(field, "expiration-time"), value.ExpirationTime, false, errs)

	if value.MaxSize <= 0 {
		maxSizeField := joinPath(field, "max-size")
		errs.add(fmt.Sprintf("%s must be greater than 0", maxSizeField))
	}

	if len(value.Methods) == 0 {
		errs.add("methods must not be empty")
	}

	for i, method := range value.Methods {
		ValidateMethod(joinPath(field, "methods", index(i)), method, false, errs)
	}
}

func ValidateTLS(_ string, mapping config.Mapping, fs afero.Fs, errs *Errors) {
	fromURL, err := mapping.GetFromURL()
	if err != nil || fromURL.Scheme != "https" {
		return
	}

	if !infratls.CAExists(fs) {
		errs.add(formatTLSError(fromURL.Host))
	}
}

func formatTLSError(host string) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "HTTPS mapping '%s' requires a local CA certificate for automatic TLS.\n\n", host)
	builder.WriteString("Generate a local CA certificate:\n")
	builder.WriteString("  uncors generate-certs\n\n")
	builder.WriteString("After generating CA, you can add it to your system's trusted certificates.")

	return builder.String()
}

func ValidateMapping(field string, value config.Mapping, fs afero.Fs, errs *Errors) {
	ValidateHost(joinPath(field, "from"), value.From, errs)
	ValidateHost(joinPath(field, "to"), value.To, errs)
	ValidateOptionsHandling(joinPath(field, "options-handling"), value.OptionsHandling, errs)
	ValidateHAR(joinPath(field, "har"), value.HAR, errs)
	ValidateTLS(field, value, fs, errs)

	for i, static := range value.Statics {
		ValidateStatic(joinPath(field, "statics", index(i)), static, fs, errs)
	}

	for i, mock := range value.Mocks {
		ValidateMock(joinPath(field, "mocks", index(i)), mock, fs, errs)
	}

	for i, glob := range value.Cache {
		ValidateCacheGlob(joinPath(field, "cache", index(i)), glob, errs)
	}

	for i, rewrite := range value.Rewrites {
		ValidateRewritingOption(joinPath(field, "rewrite", index(i)), rewrite, errs)
	}

	for i, script := range value.Scripts {
		ValidateScript(joinPath(field, "scripts", index(i)), script, fs, errs)
	}
}
