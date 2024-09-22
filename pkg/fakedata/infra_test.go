package fakedata_test

import (
	"sort"
	"testing"

	"github.com/evg4b/uncors/pkg/fakedata"
	"github.com/stretchr/testify/assert"
)

func TestGetTypes(t *testing.T) {
	expected := []string{
		"object",
		"array",
	}

	actual := fakedata.GetTypes()

	sort.Strings(expected)
	sort.Strings(actual)

	assert.Equal(t, expected, actual)
}
