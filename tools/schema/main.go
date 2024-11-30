package main

func requireNoError(err error) {
	if err != nil {
		panic(err)
	}
}

//go:generate go run .
func main() {
	uncorsJSONSchema := open("./base.json", false)

	for s, container := range loadDefinitions() {
		p(uncorsJSONSchema, refToPath(s), container.Data())
	}

	for _, container := range generateFakeDataNodes() {
		apnd(uncorsJSONSchema, "definitions.FakeDataNode.oneOf", container.Data())
	}

	write("../../schema.json", uncorsJSONSchema)
}
