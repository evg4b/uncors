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
	compiler := fakedata.NewGoFakeItGenerator()

	t.Run("seed produces deterministic results", func(t *testing.T) {
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

		expected, err := compiler.Generate(node, seed)
		require.NoError(t, err)

		for _, clonedNode := range lo.Repeat(50, node) {
			actual, err := compiler.Generate(clonedNode, seed)
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

		expected, err := compiler.Generate(node, 0)
		require.NoError(t, err)

		for _, clonedNode := range lo.Repeat(50, node) {
			actual, err := compiler.Generate(clonedNode, 0)
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
				Type: "sentence",
			},
			expect: "Who generally yourselves one lean.",
		},
		{
			name: "object",
			node: fakedata.Node{
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
			actual, err := compiler.Generate(&testCase.node, seed)
			require.NoError(t, err)

			assert.Equal(t, testCase.expect, actual)
		})
	}

	t.Run("should have base json types", func(t *testing.T) {
		cases := []struct {
			name   string
			node   fakedata.Node
			expect any
		}{
			{
				name: "number",
				node: fakedata.Node{
					Type: "number",
				},
				expect: 1321272094,
			},
			{
				name: "bool",
				node: fakedata.Node{
					Type: "bool",
				},
				expect: false,
			},
			{
				name: "string",
				node: fakedata.Node{
					Type: "string",
				},
				expect: "Necessitatibus natus numquam consequatur eos.",
			},
			{
				name: "date",
				node: fakedata.Node{
					Type: "date",
				},
				expect: "2010-07-07T19:51:28Z",
			},
		}

		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				actual, err := compiler.Generate(&testCase.node, seed)
				require.NoError(t, err)

				assert.Equal(t, testCase.expect, actual)
			})
		}
	})
}
