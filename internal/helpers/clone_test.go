package helpers_test

import (
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/helpers"

	"github.com/stretchr/testify/assert"
)

type clonableTestStruct struct{ Value string }

func (t clonableTestStruct) Clone() clonableTestStruct {
	return clonableTestStruct{Value: "Cloned:" + t.Value}
}

type nonClonableTestStruct struct{ Value string }

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

	t.Run("clonable objects", func(t *testing.T) {
		data := map[string]clonableTestStruct{
			"1": {Value: "demo"},
			"2": {Value: "demo"},
			"3": {Value: "demo"},
		}

		expected := map[string]clonableTestStruct{
			"1": {Value: "Cloned:demo"},
			"2": {Value: "Cloned:demo"},
			"3": {Value: "Cloned:demo"},
		}

		actual := helpers.CloneMap(data)

		assert.NotSame(t, &data, &actual)
		assert.EqualValues(t, &expected, &actual)
	})

	t.Run("non clonable objects", func(t *testing.T) {
		data := map[string]nonClonableTestStruct{
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
