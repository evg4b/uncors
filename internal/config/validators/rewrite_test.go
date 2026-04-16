package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fromPath = "/from/path"
	toPath   = "/to/path"
)

func TestRewritingOptionValidatorIsValidNoError(t *testing.T) {
	tests := []struct {
		name  string
		value config.RewritingOption
	}{
		{name: "valid paths and host", value: config.RewritingOption{From: fromPath, To: toPath, Host: hosts.Github.Host()}},
		{name: "no host", value: config.RewritingOption{From: fromPath, To: toPath}},
		{
			name:  "relative from path",
			value: config.RewritingOption{From: "../relative/from/path", To: toPath, Host: hosts.Github.Host()},
		},
		{
			name:  "relative to path",
			value: config.RewritingOption{From: fromPath, To: "../relative/to/path", Host: hosts.Github.Host()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errs validators.Errors
			validators.ValidateRewritingOption("testField", tt.value, &errs)
			assert.False(t, errs.HasAny())
		})
	}
}

func TestRewritingOptionValidatorIsValidWithError(t *testing.T) {
	tests := []struct {
		name  string
		value config.RewritingOption
		error string
	}{
		{
			name:  "invalid from path",
			value: config.RewritingOption{From: "", To: toPath, Host: hosts.Github.Host()},
			error: "testField.from must not be empty",
		},
		{
			name:  "invalid to path",
			value: config.RewritingOption{From: fromPath, To: "", Host: hosts.Github.Host()},
			error: "testField.to must not be empty",
		},
		{
			name:  "invalid host format",
			value: config.RewritingOption{From: fromPath, To: toPath, Host: "&&&"},
			error: "testField.host is not a valid host",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var errs validators.Errors
			validators.ValidateRewritingOption("testField", testCase.value, &errs)
			require.EqualError(t, errs, testCase.error)
		})
	}
}
