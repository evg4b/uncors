package fakedata_test

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/evg4b/uncors/pkg/fakedata"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakedataNode(t *testing.T) {
	const seed = 129

	t.Run("seed produces deterministic results", func(t *testing.T) {
		node := &fakedata.Node{
			Seed: seed,
			Type: "object",
			Properties: map[string]fakedata.Node{
				"foo": {Type: "sentence"},
				"bar": {Type: "number"},
				"baz": {
					Type: "array",
					Item: &fakedata.Node{
						Type: "object",
						Properties: map[string]fakedata.Node{
							"qux": {Type: "sentence"},
						},
					},
				},
			},
		}

		expected, err := node.Compile()
		require.NoError(t, err)

		for _, clonedNode := range lo.Repeat(50, node) {
			actual, err := clonedNode.Compile()
			require.NoError(t, err)

			assert.Equal(t, expected, actual)
		}
	})

	t.Run("produces each time new results", func(t *testing.T) {
		node := &fakedata.Node{
			Type: "object",
			Properties: map[string]fakedata.Node{
				"foo": {Type: "sentence"},
				"bar": {Type: "number"},
				"baz": {
					Type: "array",
					Item: &fakedata.Node{
						Type: "object",
						Properties: map[string]fakedata.Node{
							"qux": {Type: "sentence"},
						},
					},
				},
			},
		}

		expected, err := node.Compile()
		require.NoError(t, err)

		for _, clonedNode := range lo.Repeat(50, node) {
			actual, err := clonedNode.Compile()
			require.NoError(t, err)

			assert.NotEqual(t, expected, actual)
		}
	})

	err := gofakeit.Seed(seed)
	require.NoError(t, err)

	testCases := []struct {
		name   string
		node   fakedata.Node
		expect any
	}{
		{
			name: "sentence",
			node: fakedata.Node{
				Seed: seed,
				Type: "sentence",
			},
			expect: "Who generally yourselves one lean.",
		},
		{
			name: "object",
			node: fakedata.Node{
				Seed: seed,
				Type: "object",
				Properties: map[string]fakedata.Node{
					"foo": {Type: "sentence"},
					"bar": {Type: "number"},
				},
			},
			expect: map[string]any{
				"bar": 1321272094,
				"foo": "Thing they clarity to him.",
			},
		},
		{
			name: "array",
			node: fakedata.Node{
				Seed: seed,
				Type: "array",
				Item: &fakedata.Node{
					Type: "sentence",
				},
				Count: 3,
			},
			expect: []any{
				"Who generally yourselves one lean.",
				"Him Shakespearean there summation for.",
				"This group outside upon by.",
			},
		},
		{
			name: "array of objects",
			node: fakedata.Node{
				Seed: seed,
				Type: "array",
				Item: &fakedata.Node{
					Type: "object",
					Properties: map[string]fakedata.Node{
						"foo": {
							Type: "sentence",
						},
						"bar": {
							Type: "number",
						},
					},
				},
				Count: 2,
			},
			expect: []any{
				map[string]any{
					"bar": 1321272094,
					"foo": "Thing they clarity to him.",
				},
				map[string]any{
					"bar": -720820234,
					"foo": "We yours what for this.",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := testCase.node.Compile()
			require.NoError(t, err)

			assert.Equal(t, testCase.expect, actual)
		})
	}
}
