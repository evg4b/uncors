package schema_test

import (
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/testing/testutils"
	"github.com/evg4b/uncors/tests/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

func TestValidJsonSchema(t *testing.T) {
	testCases := schema.LoadTestCases(t, testutils.CurrentDir(t), "valid")

	jsonSchemaPath := filepath.Join(testutils.CurrentDir(t), "../../schema.json")

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			targetJSONFile := schema.TransformToJSON(t, testCase.File)

			schemaLoader := gojsonschema.NewReferenceLoader("file://" + jsonSchemaPath)
			documentLoader := gojsonschema.NewReferenceLoader("file://" + targetJSONFile)

			result, err := gojsonschema.Validate(schemaLoader, documentLoader)
			require.NoError(t, err)

			assert.Empty(t, result.Errors(), "The document is not valid")
		})
	}
}
