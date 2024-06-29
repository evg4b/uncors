package schema_test

import (
	"github.com/evg4b/uncors/tests/schema"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/testing/testutils"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

func TestValidJsonSchema(t *testing.T) {
	testdir := schema.DirPredicate("valid")

	testTempDir := t.TempDir()
	jsonSchemaPath := filepath.Join(testutils.CurrentDir(t), "../../schema.json")

	cases := []struct {
		name string
		file string
	}{
		{
			name: "minimal valid file",
			file: testdir("minimal-valid.yaml"),
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			targetJSONFile := schema.TransformToJSON(t, testTempDir, testCase.file)

			schemaLoader := gojsonschema.NewReferenceLoader("file://" + jsonSchemaPath)
			documentLoader := gojsonschema.NewReferenceLoader("file://" + targetJSONFile)

			result, err := gojsonschema.Validate(schemaLoader, documentLoader)
			require.NoError(t, err)

			assert.Empty(t, result.Errors(), "The document is not valid")
		})
	}
}
