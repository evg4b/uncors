package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Jeffail/gabs"
	"github.com/samber/lo"
)

func requireNoError(err error) {
	if err != nil {
		panic(err)
	}
}

//go:generate go run .
func main() {
	uncorsJSONSchema, err := gabs.ParseJSONFile("./base.json")
	requireNoError(err)

	for s, container := range LoadDefinitions() {
		_, err = uncorsJSONSchema.SetP(container.Data(), refToPath(s))
		requireNoError(err)
	}

	// Do something with the JSON file
	println(uncorsJSONSchema.StringIndent("", "  "))
	err = os.WriteFile("schema.json", uncorsJSONSchema.BytesIndent("", "  "), 0o644)
	requireNoError(err)
}

func LoadDefinitions() map[string]*gabs.Container {
	entries, err := os.ReadDir("definitions")
	requireNoError(err)

	files := lo.Filter(entries, func(entry os.DirEntry, _ int) bool {
		return !entry.IsDir()
	})

	storage := make(map[string]*gabs.Container, len(files))

	return lo.Reduce(files, func(storage map[string]*gabs.Container, entry os.DirEntry, _ int) map[string]*gabs.Container {
		parseJSONFile, err := gabs.ParseJSONFile(filepath.Join("definitions", entry.Name()))
		requireNoError(err)
		storage[ref(entry.Name())] = parseJSONFile
		return storage
	}, storage)
}

func ref(name string) string {
	return "#/definitions/" + strings.ReplaceAll(name, ".json", "")
}

func refToPath(name string) string {
	return strings.ReplaceAll(strings.TrimPrefix(name, "#/"), "/", ".")
}
