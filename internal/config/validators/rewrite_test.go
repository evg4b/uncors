package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewritingOptionValidator_IsValid_NoError(t *testing.T) {
	t.Run("valid host", func(t *testing.T) {
		tests := []struct {
			name  string
			field string
			value config.RewritingOption
		}{
			{
				name:  "valid paths and host",
				field: "testField",
				value: config.RewritingOption{
					From: "/from/path",
					To:   "/to/path",
					Host: "example.com",
				},
			},
			{
				name:  "invalid host",
				field: "testField",
				value: config.RewritingOption{
					From: "/from/path",
					To:   "/to/path",
					Host: "",
				},
			},
			{
				name:  "relative from path",
				field: "testField",
				value: config.RewritingOption{
					From: "../relative/from/path",
					To:   "/to/path",
					Host: "example.com",
				},
			},
			{
				name:  "relative to path",
				field: "testField",
				value: config.RewritingOption{
					From: "/from/path",
					To:   "../relative/to/path",
					Host: "example.com",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				v := &validators.RewritingOptionValidator{
					Field: tt.field,
					Value: tt.value,
				}
				errors := validate.NewErrors()
				v.IsValid(errors)

				assert.Empty(t, errors.Errors)
			})
		}
	})
}

func TestRewritingOptionValidator_IsValid_WithError(t *testing.T) {
	tests := []struct {
		name          string
		field         string
		value         config.RewritingOption
		expectedError string
	}{
		{
			name:  "invalid from path",
			field: "testField",
			value: config.RewritingOption{
				From: "",
				To:   "/to/path",
				Host: "example.com",
			},
			expectedError: "testField.from must not be empty",
		},
		{
			name:  "invalid to path",
			field: "testField",
			value: config.RewritingOption{
				From: "/from/path",
				To:   "",
				Host: "example.com",
			},
			expectedError: "testField.to must not be empty",
		},
		{
			name:  "invalid host format",
			field: "testField",
			value: config.RewritingOption{
				From: "/from/path",
				To:   "/to/path",
				Host: "&&&",
			},
			expectedError: "testField.host is not valid host",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			v := &validators.RewritingOptionValidator{
				Field: testCase.field,
				Value: testCase.value,
			}
			errors := validate.NewErrors()
			v.IsValid(errors)

			assert.NotEmpty(t, errors.Errors)
			require.EqualError(t, errors, testCase.expectedError)
		})
	}
}
