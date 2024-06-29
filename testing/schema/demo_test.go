package schema_test

import (
	"testing"

	"github.com/evg4b/uncors/testing/schema"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

func TestDemo(t *testing.T) {
	dataFile := schema.TransformToJSON(t, t.TempDir(), "demo.yaml")

	schemaLoader := gojsonschema.NewReferenceLoader("file:///Users/evg4b/Documents/uncors/schema.json")
	documentLoader := gojsonschema.NewReferenceLoader("file://" + dataFile)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	require.NoError(t, err)

	assert.True(t, result.Valid(), "The document is not valid")
}
