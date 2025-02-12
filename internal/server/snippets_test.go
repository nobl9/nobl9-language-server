package server

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/testutils"
	"github.com/nobl9/nobl9-language-server/internal/yamlastfast"
)

func TestSnippetsComplete(t *testing.T) {
	testFilesDir := filepath.Join(testutils.FindModuleRoot(), "internal", "server", "testdata", "snippets")
	getTestFileURI := func(name string) messages.TextDocumentIdentifier {
		return messages.TextDocumentIdentifier{URI: filepath.Join(testFilesDir, name)}
	}

	fileSystem := newFilesystem()
	registerTestFiles(t, fileSystem, testFilesDir)

	snippets := newSnippetsProvider()

	srv := Server{
		files:                   fileSystem,
		snippetsProvider:        snippets,
		completionProvider:      mockCompletionProvider{},
		pathsCompletionProvider: mockPathsCompletionProvider{},
	}

	tests := map[string]struct {
		Params      messages.CompletionParams
		Path        string
		Line        int
		FirstResult messages.CompletionItem
	}{
		"document without predecessor": {
			Params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("snippet.yaml"),
					Position: messages.Position{
						Line:      0,
						Character: 0,
					},
				},
			},
			FirstResult: messages.CompletionItem{
				Label: "project",
				InsertText: `apiVersion: n9/v1alpha
kind: Project
metadata:
  displayName: $1
  name: $1
spec:
  description: $2
`,
			},
		},
		"document with preceding node": {
			Params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("snippet.yaml"),
					Position: messages.Position{
						Line:      3,
						Character: 0,
					},
				},
			},
			FirstResult: messages.CompletionItem{
				Label: "project",
				InsertText: `---
apiVersion: n9/v1alpha
kind: Project
metadata:
  displayName: $1
  name: $1
spec:
  description: $2
`,
			},
		},
		"list of objects with indent at the cursor": {
			Params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("snippet.yaml"),
					Position: messages.Position{
						Line:      6,
						Character: 0,
					},
				},
			},
			FirstResult: messages.CompletionItem{
				Label: "project",
				InsertText: `apiVersion: n9/v1alpha
  kind: Project
  metadata:
    displayName: $1
    name: $1
  spec:
    description: $2
`,
			},
		},
		"list of objects without indent": {
			Params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("snippet.yaml"),
					Position: messages.Position{
						Line:      9,
						Character: 0,
					},
				},
			},
			FirstResult: messages.CompletionItem{
				Label: "project",
				InsertText: `- apiVersion: n9/v1alpha
  kind: Project
  metadata:
    displayName: $1
    name: $1
  spec:
    description: $2
`,
			},
		},
		"document in between separator and another doc": {
			Params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("snippet.yaml"),
					Position: messages.Position{
						Line:      11,
						Character: 0,
					},
				},
			},
			FirstResult: messages.CompletionItem{
				Label: "project",
				InsertText: `apiVersion: n9/v1alpha
kind: Project
metadata:
  displayName: $1
  name: $1
spec:
  description: $2
`,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := srv.handleCompletion(context.Background(), test.Params)
			require.NoError(t, err)
			require.NotNil(t, result)

			items := result.([]messages.CompletionItem)
			require.Greater(t, len(items), 1)

			test.FirstResult.Kind = messages.SnippetCompletion
			test.FirstResult.InsertTextFormat = messages.SnippetTextFormat
			assert.Equal(t, test.FirstResult, items[0])
		})
	}
}

type mockCompletionProvider struct{}

func (m mockCompletionProvider) Complete(messages.CompletionContext, manifest.Kind, string) []messages.CompletionItem {
	return nil
}

type mockPathsCompletionProvider struct{}

func (m mockPathsCompletionProvider) Complete(
	manifest.Kind,
	*yamlastfast.Line,
	messages.Position,
) []messages.CompletionItem {
	return nil
}
