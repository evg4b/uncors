package main

import (
	"os"
	"strings"

	"github.com/Jeffail/gabs"
)

func ref(name string) string {
	return "#/definitions/" + strings.ReplaceAll(name, ".json", "")
}

func refToPath(name string) string {
	return strings.ReplaceAll(strings.TrimPrefix(name, "#/"), "/", ".")
}

func p(object *gabs.Container, path string, value any) {
	_, err := object.SetP(value, path)
	requireNoError(err)
}

func f(path string) *gabs.Container {
	return open(path, true)
}

func open(path string, clean bool) *gabs.Container {
	uncorsJSONSchema, err := gabs.ParseJSONFile(path)
	requireNoError(err)
	if clean {
		err = uncorsJSONSchema.Delete("$schema")
		requireNoError(err)
	}

	return uncorsJSONSchema
}

func write(path string, object *gabs.Container) {
	err := os.WriteFile(path, object.BytesIndent("", "  "), os.ModePerm) //nolint:gosec
	requireNoError(err)
}
