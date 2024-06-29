package schema

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TransformToJSON(t *testing.T, dir string, file string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("Failed to get caller information")
	}

	yamlFilePath := filepath.Join(filepath.Dir(filename), file)
	jsonFilePath := path.Join(dir, strings.Replace(file, ".yaml", ".json", 1))

	yamlFile, err := os.OpenFile(yamlFilePath, os.O_RDONLY, os.ModePerm)
	require.NoError(t, err, "Failed to open file: %v", err)
	defer yamlFile.Close()

	jsonFile, err := os.OpenFile(jsonFilePath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	require.NoError(t, err, "Failed to open file: %v", err)
	defer jsonFile.Close()

	var data any
	err = yaml.NewDecoder(yamlFile).Decode(&data)
	require.NoError(t, err, "Failed to decode yaml: %v", err)

	err = json.NewEncoder(jsonFile).Encode(data)
	require.NoError(t, err, "Failed to encode json: %v", err)

	return jsonFilePath
}
