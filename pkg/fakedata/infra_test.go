package fakedata_test

import (
	"sort"
	"testing"

	"github.com/evg4b/uncors/pkg/fakedata"
	"github.com/stretchr/testify/assert"
)

func TestGetTypes(t *testing.T) {
	expected := []string{
		// Inner types
		"object",
		"array",
		// number
		"number",
		"int",
		"intn",
		"int8",
		"int16",
		"int32",
		"int64",
		"uint",
		"uintn",
		"uint8",
		"uint16",
		"uint32",
		"uint64",
		"float32",
		"float32range",
		"float64",
		"float64range",
	}

	actual := fakedata.GetTypes()

	sort.Strings(expected)
	sort.Strings(actual)

	assert.Equal(t, expected, actual)
}
