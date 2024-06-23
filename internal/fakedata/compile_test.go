package fakedata_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/fakedata"
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
			name: "object",
			node: fakedata.Node{
				Seed: seed,
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
			expect: map[string]any{
				"foo": "Who generally yourselves one lean.",
				"bar": 1089124290,
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
