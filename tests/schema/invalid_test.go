package schema_test

import (
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/tests/schema"

	"github.com/evg4b/uncors/testing/testutils"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

func TestInvalidJsonSchema(t *testing.T) {
	testCases := schema.LoadTestCasesWithErrors(t, testutils.CurrentDir(t), "invalid")

	jsonSchemaPath := filepath.Join(testutils.CurrentDir(t), "../../schema.json")

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			targetJSONFile := schema.TransformToJSON(t, testCase.File)

			schemaLoader := gojsonschema.NewReferenceLoader("file://" + jsonSchemaPath)
			documentLoader := gojsonschema.NewReferenceLoader("file://" + targetJSONFile)

			result, err := gojsonschema.Validate(schemaLoader, documentLoader)
			require.NoError(t, err)

			errors := lo.Map(result.Errors(), func(err gojsonschema.ResultError, _ int) string {
				return err.String()
			})

			assert.Equal(t, testCase.Errors, errors, "The errors are not as expected")
		})
	}
}
