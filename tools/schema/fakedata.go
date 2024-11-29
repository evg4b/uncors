package main

import "github.com/Jeffail/gabs"

func generateFakeDataNodes() []*gabs.Container {

	return []*gabs.Container{
		f("./fakedata/object.json"),
		f("./fakedata/array.json"),
	}
}
