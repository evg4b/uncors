package fakedata_test

import (
	"testing"

	"github.com/evg4b/uncors/pkg/fakedata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDemo(t *testing.T) {
	const seed = 129

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
				"foo": "Who generally yourselves one lean.",
				"bar": 1089124290,
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
					"bar": 1089124290,
					"foo": "Who generally yourselves one lean.",
				},
				map[string]any{
					"bar": -1123283869,
					"foo": "Beyond we yours what for.",
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
