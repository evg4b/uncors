package schema

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TransformToJSON(t *testing.T, yamlFilePath string) string {
	t.Helper()

	jsonFilePath := filepath.Join(t.TempDir(), strings.Replace(filepath.Base(yamlFilePath), ".yaml", ".json", 1))

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

type TestCase struct {
	Name   string
	File   string
	Errors []string
}

func readErrors(t *testing.T, filePath string) []string {
	t.Helper()

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)

	return strings.Split(string(content), "\n")
}

func LoadTestCases(t *testing.T, parts ...string) []TestCase {
	return loadTestCasesInternal(t, false, parts...)
}

func LoadTestCasesWithErrors(t *testing.T, parts ...string) []TestCase {
	return loadTestCasesInternal(t, true, parts...)
}

func loadTestCasesInternal(t *testing.T, errors bool, parts ...string) []TestCase {
	t.Helper()
	dir := filepath.Join(parts...)

	testCases := make([]TestCase, 0, 30) //nolint:mnd
	err := fs.WalkDir(os.DirFS(dir), ".", func(path string, entry fs.DirEntry, err error) error {
		require.NoError(t, err)
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			var errorsArray []string
			if errors {
				errorsArray = readErrors(t, filepath.Join(dir, entry.Name()+".errors"))
			}

			testCases = append(testCases, TestCase{
				Name: strings.ReplaceAll(
					strings.ReplaceAll(path, ".yaml", ""),
					"-",
					" ",
				),
				File:   filepath.Join(dir, path),
				Errors: errorsArray,
			})
		}

		return nil
	})
	require.NoError(t, err)

	return testCases
}
