package server

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/sdkdocs"
	"github.com/nobl9/nobl9-language-server/internal/testutils"
)

func TestServerComplete(t *testing.T) {
	testFilesDir := filepath.Join(testutils.FindModuleRoot(), "internal", "server", "testdata", "completion")
	getTestFileURI := func(name string) messages.TextDocumentIdentifier {
		return messages.TextDocumentIdentifier{URI: filepath.Join(testFilesDir, name)}
	}

	fileSystem := newFilesystem()
	registerTestFiles(t, fileSystem, testFilesDir)

	docs, err := sdkdocs.New()
	require.NoError(t, err)
	provider := newPathsCompletionProvider(docs)

	srv := Server{
		files:                   fileSystem,
		completionProvider:      &completionProvidersRegistry{},
		snippetsProvider:        &snippetsProvider{},
		pathsCompletionProvider: provider,
	}

	tests := map[string]struct {
		params   messages.CompletionParams
		expected []messages.CompletionItem
	}{
		"agent - complete root": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("agent.yaml"),
					Position: messages.Position{
						Line:      3,
						Character: 2,
					},
				},
			},
			expected: []messages.CompletionItem{
				{Label: "apiVersion", Kind: messages.PropertyCompletion},
				{Label: "kind", Kind: messages.PropertyCompletion},
				{Label: "metadata", Kind: messages.PropertyCompletion},
				{Label: "spec", Kind: messages.PropertyCompletion},
			},
		},
		"agent - complete metadata": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("agent.yaml"),
					Position: messages.Position{
						Line:      4,
						Character: 3,
					},
				},
			},
			expected: []messages.CompletionItem{
				{Label: "name", Kind: messages.PropertyCompletion},
				{Label: "displayName", Kind: messages.PropertyCompletion},
				{Label: "project", Kind: messages.PropertyCompletion},
			},
		},
		"agent - do not complete metadata if cursor is on value": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("agent.yaml"),
					Position: messages.Position{
						Line:      2,
						Character: 6,
					},
				},
			},
			expected: []messages.CompletionItem{},
		},
		"barebones - complete root": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("barebones.yaml"),
					Position: messages.Position{
						Line:      0,
						Character: 1,
					},
				},
			},
			expected: []messages.CompletionItem{
				{Label: "apiVersion", Kind: messages.PropertyCompletion},
				{Label: "kind", Kind: messages.PropertyCompletion},
				{Label: "metadata", Kind: messages.PropertyCompletion},
				{Label: "spec", Kind: messages.PropertyCompletion},
			},
		},
		"unfinished metadata - complete metadata": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("unfinished-metadata.yaml"),
					Position: messages.Position{
						Line:      4,
						Character: 2,
					},
				},
			},
			expected: []messages.CompletionItem{
				{Label: "name", Kind: messages.PropertyCompletion},
				{Label: "displayName", Kind: messages.PropertyCompletion},
				{Label: "project", Kind: messages.PropertyCompletion},
			},
		},
		"unfinished metadata - complete metadata complex": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("unfinished-metadata-complex.yaml"),
					Position: messages.Position{
						Line:      4,
						Character: 3,
					},
				},
			},
			expected: []messages.CompletionItem{
				{Label: "name", Kind: messages.PropertyCompletion},
				{Label: "displayName", Kind: messages.PropertyCompletion},
				{Label: "project", Kind: messages.PropertyCompletion},
			},
		},
		"root path with faulty standard YAML AST": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("service-and-agent.yaml"),
					Position: messages.Position{
						Line:      22,
						Character: 1,
					},
				},
			},
			expected: []messages.CompletionItem{
				{Label: "apiVersion", Kind: messages.PropertyCompletion},
				{Label: "kind", Kind: messages.PropertyCompletion},
				{Label: "metadata", Kind: messages.PropertyCompletion},
				{Label: "spec", Kind: messages.PropertyCompletion},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			items, err := srv.handleCompletion(context.Background(), test.params)
			require.NoError(t, err)
			if len(test.expected) > 0 {
				require.NotNil(t, items)
			}
			assert.ElementsMatch(t, test.expected, items)
		})
	}
}

func Test_normalizeRootArrayPath(t *testing.T) {
	tests := []struct {
		Input    string
		Expected string
	}{
		{Input: "$", Expected: "$"},
		{Input: "$[1]", Expected: "$"},
		{Input: "$[10]", Expected: "$"},
		{Input: "$.[3]", Expected: "$.[3]"},
		{Input: "$[5].A.B", Expected: "$.A.B"},
	}
	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			assert.Equal(t, test.Expected, normalizeRootPath(test.Input))
		})
	}
}

func registerTestFiles(t *testing.T, fs *filesystem, testFileDir string) {
	entries, err := os.ReadDir(testFileDir)
	require.NoError(t, err)
	for _, entry := range entries {
		require.False(t, entry.IsDir())
		path := filepath.Join(testFileDir, entry.Name())
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		err = fs.OpenFile(context.Background(), path, string(data), 1)
		require.NoError(t, err)
	}
}
