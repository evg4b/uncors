package main

import "github.com/evg4b/uncors/pkg/fakedata"

func main() {
	for i, s := range fakedata.GetTypes() {
		println(i, s)
	}
}
