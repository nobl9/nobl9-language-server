package yamlast

import (
	"embed"
	_ "embed"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata
var testParseInput embed.FS

func TestParse(t *testing.T) {
	tests := map[string][]Node{
		"document.yaml": {
			{StartLine: 1, EndLine: 4},
		},
		"two-documents.yaml": {
			{StartLine: 1, EndLine: 4},
			{StartLine: 6, EndLine: 10},
		},
		"single-element-sequence.yaml": {
			{StartLine: 1, EndLine: 5},
		},
		"sequence.yaml": {
			{StartLine: 1, EndLine: 4},
			{StartLine: 5, EndLine: 8},
			{StartLine: 9, EndLine: 13},
		},
		"sequence-with-document-separator.yaml": {
			{StartLine: 2, EndLine: 5},
			{StartLine: 6, EndLine: 9},
			{StartLine: 10, EndLine: 13},
		},
		"two-sequence-documents.yaml": {
			{StartLine: 1, EndLine: 4},
			{StartLine: 5, EndLine: 8},
			{StartLine: 10, EndLine: 13},
			{StartLine: 14, EndLine: 18},
		},
		"mixed-documents.yaml": {
			{StartLine: 1, EndLine: 4},
			{StartLine: 6, EndLine: 9},
			{StartLine: 10, EndLine: 13},
			{StartLine: 15, EndLine: 18},
			{StartLine: 20, EndLine: 24},
		},
		"weird-comments.yaml": {
			{StartLine: 2, EndLine: 32},
		},
	}
	for filename, docs := range tests {
		t.Run(filename, func(t *testing.T) {
			file, err := Parse(readTestInput(t, filename))
			require.NoError(t, err)
			require.Len(t, file.Nodes, len(docs), "Document count mismatch")
			for i := range file.Nodes {
				actualDoc := file.Nodes[i]
				expectedDoc := docs[i]
				assert.Equal(t, expectedDoc.StartLine, actualDoc.StartLine, "start line mismatch")
				assert.Equal(t, expectedDoc.EndLine, actualDoc.EndLine, "End line mismatch")
			}
		})
	}
}

func readTestInput(t *testing.T, file string) string {
	t.Helper()
	data, err := testParseInput.ReadFile(filepath.Join("testdata", file))
	require.NoError(t, err)
	return string(data)
}
