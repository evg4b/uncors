package schema_test

import (
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/testing/testutils"
	"github.com/evg4b/uncors/tests/schema"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

func loadUncorsSchema(t *testing.T) gojsonschema.JSONLoader {
	jsonSchemaPath := filepath.Join(testutils.CurrentDir(t), "../../schema.json")

	return gojsonschema.NewReferenceLoader("file://" + jsonSchemaPath)
}

func loadFileSchema(t *testing.T, file string) gojsonschema.JSONLoader {
	targetJSONFile := schema.TransformToJSON(t, file)

	return gojsonschema.NewReferenceLoader("file://" + targetJSONFile)
}

func TestJsonSchema(t *testing.T) {
	schemaLoader := gojsonschema.NewReferenceLoader("http://json-schema.org/draft-07/schema#")
	documentLoader := loadUncorsSchema(t)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	require.NoError(t, err)

	assert.Empty(t, result.Errors(), "The document is not valid")
}

func TestInvalidJsonSchema(t *testing.T) {
	testCases := schema.LoadTestCasesWithErrors(t, testutils.CurrentDir(t), "invalid")
	schemaLoader := loadUncorsSchema(t)

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			documentLoader := loadFileSchema(t, testCase.File)

			result, err := gojsonschema.Validate(schemaLoader, documentLoader)
			require.NoError(t, err)

			errors := lo.Map(result.Errors(), func(err gojsonschema.ResultError, _ int) string {
				return err.String()
			})

			assert.Equal(t, testCase.Errors, errors, "The errors are not as expected")
		})
	}
}

func TestValidJsonSchema(t *testing.T) {
	testCases := schema.LoadTestCases(t, testutils.CurrentDir(t), "valid")
	schemaLoader := loadUncorsSchema(t)

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			documentLoader := loadFileSchema(t, testCase.File)

			result, err := gojsonschema.Validate(schemaLoader, documentLoader)
			require.NoError(t, err)

			assert.Empty(t, result.Errors(), "The document is not valid")
		})
	}
}
