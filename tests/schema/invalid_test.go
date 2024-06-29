package schema_test

import (
	"github.com/evg4b/uncors/tests/schema"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/testing/testutils"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

func TestInvalidJsonSchema(t *testing.T) {
	testdir := schema.DirPredicate("invalid")

	testTempDir := t.TempDir()
	jsonSchemaPath := filepath.Join(testutils.CurrentDir(t), "../../schema.json")

	cases := []struct {
		name   string
		file   string
		errors []string
	}{
		{
			name: "empty mappings",
			file: testdir("empty-mappings.yaml"),
			errors: []string{
				"mappings: Array must have at least 1 items",
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			targetJSONFile := schema.TransformToJSON(t, testTempDir, testCase.file)

			schemaLoader := gojsonschema.NewReferenceLoader("file://" + jsonSchemaPath)
			documentLoader := gojsonschema.NewReferenceLoader("file://" + targetJSONFile)

			result, err := gojsonschema.Validate(schemaLoader, documentLoader)
			require.NoError(t, err)

			errors := lo.Map(result.Errors(), func(err gojsonschema.ResultError, _ int) string {
				return err.String()
			})

			assert.Equal(t, testCase.errors, errors, "The errors are not as expected")
		})
	}
}
