package helpers_test

import (
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/stretchr/testify/assert"
)

type cloneableTestStruct struct{ Value string }

func (t *cloneableTestStruct) Clone() cloneableTestStruct {
	return cloneableTestStruct{Value: "Cloned:" + t.Value}
}

type nonCloneableTestStruct struct{ Value string }

func TestCloneMap(t *testing.T) {
	t.Run("base types", func(t *testing.T) {
		t.Run("clone map[string]string", func(t *testing.T) {
			assertClone(t, map[string]string{
				"1": "2",
				"2": "3",
				"3": "4",
				"4": "1",
			})
		})

		t.Run("clone map[string]int", func(t *testing.T) {
			assertClone(t, map[string]int{
				"1": 2,
				"2": 3,
				"3": 4,
				"4": 1,
			})
		})

		t.Run("clone map[string]any", func(t *testing.T) {
			assertClone(t, map[string]any{
				"1": 2,
				"2": "2",
				"3": time.Hour,
				"4": []int{1, 2, 3},
			})
		})

		t.Run("clone map[int]string", func(t *testing.T) {
			assertClone(t, map[int]string{
				1: "2",
				2: "3",
				3: "4",
				4: "1",
			})
		})

		t.Run("clone map[int]int", func(t *testing.T) {
			assertClone(t, map[int]int{
				1: 2,
				2: 3,
				3: 4,
				4: 1,
			})
		})

		t.Run("clone map[int]any", func(t *testing.T) {
			assertClone(t, map[int]any{
				1: 2,
				2: "2",
				3: time.Hour,
				4: []int{1, 2, 3},
			})
		})
	})

	t.Run("cloneable objects", func(t *testing.T) {
		data := map[string]cloneableTestStruct{
			"1": {Value: "property 1"},
			"2": {Value: "property 2"},
			"3": {Value: "property 3"},
		}

		expected := map[string]cloneableTestStruct{
			"1": {Value: "Cloned:property 1"},
			"2": {Value: "Cloned:property 2"},
			"3": {Value: "Cloned:property 3"},
		}

		actual := helpers.CloneMap(data)

		assert.NotSame(t, &data, &actual)
		assert.EqualValues(t, &expected, &actual)
	})

	t.Run("non cloneable objects", func(t *testing.T) {
		data := map[string]nonCloneableTestStruct{
			"1": {Value: "demo"},
			"2": {Value: "demo"},
			"3": {Value: "demo"},
		}

		actual := helpers.CloneMap(data)

		assert.NotSame(t, &data, &actual)
		assert.EqualValues(t, &data, &actual)
	})

	t.Run("nil", func(t *testing.T) {
		actual := helpers.CloneMap[string, string](nil)

		assert.Nil(t, actual)
	})
}

func assertClone[K comparable, V any](t *testing.T, data map[K]V) {
	t.Helper()

	actual := helpers.CloneMap(data)

	assert.EqualValues(t, data, actual)
	assert.NotSame(t, &data, &actual)
}
