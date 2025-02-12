package yamlast

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/find-path-input.yaml
var findPathTestInput string

func TestNodeFind(t *testing.T) {
	file, err := Parse(findPathTestInput)
	require.NoError(t, err)

	tests := []struct {
		Line int
		Path string
	}{
		{
			Line: 60,
			Path: "$.spec.objectives[1].rawMetric.query.datadog.query",
		},
		{
			Line: 30,
			Path: "$.spec.description",
		},
		{
			Line: 4,
			Path: "$.metadata",
		},
		{
			Line: 2,
			Path: "$.apiVersion",
		},
		{
			Line: 24,
			Path: "$.metadata.labels",
		},
	}

	for _, test := range tests {
		t.Run(test.Path, func(t *testing.T) {
			node := findNode(file.Nodes, test.Line)
			require.NotEmpty(t, node)
			found, err := node.Find(test.Line)
			require.NoError(t, err)
			require.NotEmpty(t, found)
			assert.Equal(t, test.Path, found.GetPath())
		})
	}
}

func findNode(nodes []*Node, line int) *Node {
	for _, node := range nodes {
		if line >= node.StartLine && line <= node.EndLine {
			return node
		}
	}
	return nil
}
