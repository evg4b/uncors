package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TransformToJSON(t *testing.T, dir string, file string) string {
	t.Helper()
	yamlFilePath := filepath.Join(testutils.CurrentDir(t), file)
	jsonFilePath := filepath.Join(dir, strings.Replace(filepath.Base(file), ".yaml", ".json", 1))

	yamlFile, err := os.OpenFile(yamlFilePath, os.O_RDONLY, os.ModePerm)
	require.NoError(t, err, "Failed to open file: %v", err)
	defer yamlFile.Close()

	jsonFile, err := os.OpenFile(jsonFilePath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	require.NoError(t, err, "Failed to open file: %v", err)
	defer yamlFile.Close()

	var data any
	err = yaml.NewDecoder(yamlFile).Decode(&data)
	require.NoError(t, err, "Failed to decode yaml: %v", err)

	err = json.NewEncoder(jsonFile).Encode(data)
	require.NoError(t, err, "Failed to encode json: %v", err)

	return jsonFilePath
}

func DirPredicate(dir string) func(string) string {
	return func(file string) string {
		return filepath.Join(dir, file)
	}
}
