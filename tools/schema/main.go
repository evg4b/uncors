package main

import (
	"os"
)

func requireNoError(err error) {
	if err != nil {
		panic(err)
	}
}

//go:generate go run .
func main() {
	uncorsJSONSchema := open("./base.json", false)

	//for s, container := range loadDefinitions() {
	//	_, err = uncorsJSONSchema.SetP(container.Data(), refToPath(s))
	//	requireNoError(err)
	//}

	for _, container := range generateFakeDataNodes() {
		uncorsJSONSchema.ArrayAppendP(container.Data(), "definitions.FakeDataNode.oneOf")
	}

	// Do something with the JSON file
	println(uncorsJSONSchema.StringIndent("", "  "))
	err := os.WriteFile("schema.json", uncorsJSONSchema.BytesIndent("", "  "), os.ModePerm) //nolint:gosec
	requireNoError(err)
}
