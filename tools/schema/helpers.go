package main

import (
	"github.com/Jeffail/gabs"
	"strings"
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

func open(path string, clear bool) *gabs.Container {
	uncorsJSONSchema, err := gabs.ParseJSONFile(path)
	requireNoError(err)
	if clear {
		err = uncorsJSONSchema.Delete("$schema")
		requireNoError(err)
	}

	return uncorsJSONSchema
}
