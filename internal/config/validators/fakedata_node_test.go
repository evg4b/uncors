package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/evg4b/uncors/pkg/fakedata"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
)

func TestFakedataNodeValidator(t *testing.T) {
	const field = "fake"

	t.Run("valid options", func(t *testing.T) {
		tests := []struct {
			name  string
			value *fakedata.Node
			root  bool
		}{
			{
				name: "empty object",
				value: &fakedata.Node{
					Type:       "object",
					Properties: map[string]fakedata.Node{},
				},
				root: true,
			},
			{
				name: "empty array",
				value: &fakedata.Node{
					Type: "array",
					Item: &fakedata.Node{
						Type: "number",
					},
				},
				root: false,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				errors := validate.Validate(&validators.FakedataNodeValidator{
					Field: field,
					Value: testCase.value,
					Root:  testCase.root,
				})

				assert.False(t, errors.HasAny())
			})
		}
	})

	t.Run("invalid options", func(t *testing.T) {
		tests := []struct {
			name  string
			value *fakedata.Node
			root  bool
			err   string
		}{
			{
				name:  "unknown fake data type",
				value: &fakedata.Node{Type: "unknown"},
				root:  true,
				err:   "'unknown' is not a valid option",
			},
			{
				name:  "unknown fake data type",
				value: &fakedata.Node{Type: "number", Seed: 1},
				root:  false,
				err:   "property 'seed' is not allowed in nested nodes",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				errors := validate.Validate(&validators.FakedataNodeValidator{
					Field: field,
					Value: testCase.value,
					Root:  testCase.root,
				})

				assert.EqualError(t, errors, testCase.err)
			})
		}
	})
}
