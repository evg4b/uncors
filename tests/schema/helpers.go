package schema

import (
	"encoding/json"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TransformToJSON(t *testing.T, yamlFilePath string) string {
	t.Helper()

	fs := afero.NewOsFs()
	jsonFilePath := filepath.Join(t.TempDir(), strings.Replace(filepath.Base(yamlFilePath), ".yaml", ".json", 1))

	yamlFile, err := fs.Open(yamlFilePath)
	require.NoError(t, err, "Failed to open file: %v", err)
	defer yamlFile.Close()

	jsonFile, err := fs.Create(jsonFilePath)
	require.NoError(t, err, "Failed to create file: %v", err)
	defer jsonFile.Close()

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

	fs := afero.NewOsFs()
	content, err := afero.ReadFile(fs, filePath)
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

	afs := afero.NewOsFs()
	testCases := make([]TestCase, 0, 30) //nolint:mnd
	err := afero.Walk(afs, dir, func(path string, info fs.FileInfo, err error) error {
		require.NoError(t, err)
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".yaml") {
			var errorsArray []string
			if errors {
				errorsArray = readErrors(t, path+".errors")
			}

			relPath, err := filepath.Rel(dir, path)
			require.NoError(t, err)

			testCases = append(testCases, TestCase{
				Name: strings.ReplaceAll(
					strings.ReplaceAll(relPath, ".yaml", ""),
					"-",
					" ",
				),
				File:   path,
				Errors: errorsArray,
			})
		}

		return nil
	})
	require.NoError(t, err)

	return testCases
}
