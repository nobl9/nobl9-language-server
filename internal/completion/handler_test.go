package completion

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/sdkdocs"
	"github.com/nobl9/nobl9-language-server/internal/testutils"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	testDir := filepath.Join(testutils.FindModuleRoot(), "internal", "completion", "testdata")
	getTestFileURI := func(name string) messages.TextDocumentIdentifier {
		return messages.TextDocumentIdentifier{URI: filepath.Join(testDir, name)}
	}

	fileSystem := files.NewFS()
	testutils.RegisterTestFiles(t, fileSystem, testDir)

	docs, err := sdkdocs.New()
	require.NoError(t, err)
	repo := mockObjectsRepo{
		projectNames: []string{"project1", "project2"},
	}

	handler := &Handler{
		files: fileSystem,
		providers: []providerInterface{
			NewReferencesCompletionProvider(NewObjectsRefProvider(repo)),
			NewKeysCompletionProvider(docs),
			NewValuesCompletionProvider(docs),
			NewSnippetsProvider(),
		},
	}

	tests := map[string]struct {
		params           messages.CompletionParams
		expected         []messages.CompletionItem
		firstSnippetItem *messages.CompletionItem
		ignoreSnippets   bool
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
			expected: rootPathCompletionItems,
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
			expected: metadataPathCompletionItems,
		},
		"agent - do not complete metadata if cursor is on value": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("agent.yaml"),
					Position: messages.Position{
						Line:      4,
						Character: 13,
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
			expected:       rootPathCompletionItems,
			ignoreSnippets: true,
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
			expected: metadataPathCompletionItems,
		},
		"unfinished metadata - complete metadata complex": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("unfinished-metadata-complex.yaml"),
					Position: messages.Position{
						Line:      4,
						Character: 2,
					},
				},
			},
			expected: metadataPathCompletionItems,
		},
		"root path with faulty standard YAML AST": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("service-and-agent.yaml"),
					Position: messages.Position{
						Line:      22,
						Character: 0,
					},
				},
			},
			expected:       rootPathCompletionItems,
			ignoreSnippets: true,
		},
		"complete apiVersion": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("complete-apiversion.yaml"),
					Position: messages.Position{
						Line:      0,
						Character: 13,
					},
				},
			},
			expected: apiVersionCompletionItems,
		},
		"complete kind": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("complete-kind.yaml"),
					Position: messages.Position{
						Line:      1,
						Character: 6,
					},
				},
			},
			expected: kindNameCompletionItems,
		},
		"complete kind - full object": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("complete-kind-full-object.yaml"),
					Position: messages.Position{
						Line:      1,
						Character: 6,
					},
				},
			},
			expected: kindNameCompletionItems,
		},
		"complete SLO budgeting method": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("complete-budgeting-method.yaml"),
					Position: messages.Position{
						Line:      7,
						Character: 19,
					},
				},
			},
			expected: []messages.CompletionItem{
				{Label: "Timeslices", Kind: messages.ValueCompletion},
				{Label: "Occurrences", Kind: messages.ValueCompletion},
			},
		},
		"complete project names - value start": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("complete-project-names.yaml"),
					Position: messages.Position{
						Line:      4,
						Character: 11,
					},
				},
			},
			expected: []messages.CompletionItem{
				{Label: "project1", Kind: messages.ReferenceCompletion},
				{Label: "project2", Kind: messages.ReferenceCompletion},
			},
		},
		"complete project names - value end": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("complete-project-names.yaml"),
					Position: messages.Position{
						Line:      4,
						Character: 12,
					},
				},
			},
			expected: []messages.CompletionItem{
				{Label: "project1", Kind: messages.ReferenceCompletion},
				{Label: "project2", Kind: messages.ReferenceCompletion},
			},
		},
		"document without predecessor": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("snippet.yaml"),
					Position: messages.Position{
						Line:      0,
						Character: 0,
					},
				},
			},
			expected: rootPathCompletionItems,
			firstSnippetItem: &messages.CompletionItem{
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
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("snippet.yaml"),
					Position: messages.Position{
						Line:      3,
						Character: 0,
					},
				},
			},
			expected: rootPathCompletionItems,
			firstSnippetItem: &messages.CompletionItem{
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
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("snippet.yaml"),
					Position: messages.Position{
						Line:      6,
						Character: 0,
					},
				},
			},
			expected: rootPathCompletionItems,
			firstSnippetItem: &messages.CompletionItem{
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
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("snippet.yaml"),
					Position: messages.Position{
						Line:      9,
						Character: 0,
					},
				},
			},
			expected: rootPathCompletionItems,
			firstSnippetItem: &messages.CompletionItem{
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
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("snippet.yaml"),
					Position: messages.Position{
						Line:      11,
						Character: 0,
					},
				},
			},
			expected: rootPathCompletionItems,
			firstSnippetItem: &messages.CompletionItem{
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
		"complete composite field": {
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("composite.yaml"),
					Position: messages.Position{
						Line:      33,
						Character: 10,
					},
				},
			},
			expected: []messages.CompletionItem{
				{
					Label:            "objectives[*]",
					Kind:             messages.PropertyCompletion,
					InsertText:       "objectives:\n          - ",
					InsertTextFormat: messages.PlainTextTextFormat,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := handler.Handle(context.Background(), test.params)
			require.NoError(t, err)
			items := result.([]messages.CompletionItem)
			if len(test.expected) > 0 {
				require.NotNil(t, items)
			}

			if test.ignoreSnippets {
				items = slices.DeleteFunc(items, func(item messages.CompletionItem) bool {
					return item.Kind == messages.SnippetCompletion
				})
			}

			if test.firstSnippetItem != nil {
				var firstSnippetItem messages.CompletionItem
				for _, item := range items {
					if item.Kind == messages.SnippetCompletion {
						firstSnippetItem = item
						break
					}
				}
				require.NotEmpty(t, firstSnippetItem)
				items = slices.DeleteFunc(items, func(item messages.CompletionItem) bool {
					return item.Kind == messages.SnippetCompletion
				})
				test.firstSnippetItem.Kind = messages.SnippetCompletion
				test.firstSnippetItem.InsertTextFormat = messages.SnippetTextFormat
				assert.Equal(t, *test.firstSnippetItem, firstSnippetItem)
			}

			assert.ElementsMatch(t, test.expected, items)
		})
	}
}

var (
	rootPathCompletionItems = []messages.CompletionItem{
		{
			Label:            "apiVersion",
			Kind:             messages.PropertyCompletion,
			InsertText:       "apiVersion: ",
			InsertTextFormat: messages.PlainTextTextFormat,
		},
		{
			Label:            "kind",
			Kind:             messages.PropertyCompletion,
			InsertText:       "kind: ",
			InsertTextFormat: messages.PlainTextTextFormat,
		},
		{
			Label:            "metadata",
			Kind:             messages.PropertyCompletion,
			InsertText:       "metadata:\n  ",
			InsertTextFormat: messages.PlainTextTextFormat,
		},
		{
			Label:            "spec",
			Kind:             messages.PropertyCompletion,
			InsertText:       "spec:\n  ",
			InsertTextFormat: messages.PlainTextTextFormat,
		},
	}
	metadataPathCompletionItems = []messages.CompletionItem{
		{
			Label:            "name",
			Kind:             messages.PropertyCompletion,
			InsertText:       "name: ",
			InsertTextFormat: messages.PlainTextTextFormat,
		},
		{
			Label:            "displayName",
			Kind:             messages.PropertyCompletion,
			InsertText:       "displayName: ",
			InsertTextFormat: messages.PlainTextTextFormat,
		},
		{
			Label:            "project",
			Kind:             messages.PropertyCompletion,
			InsertText:       "project: ",
			InsertTextFormat: messages.PlainTextTextFormat,
		},
	}
	kindNameCompletionItems = func() (items []messages.CompletionItem) {
		for _, version := range manifest.KindNames() {
			items = append(items, messages.CompletionItem{
				Label: version,
				Kind:  messages.ValueCompletion,
			})
		}
		return items
	}()
	apiVersionCompletionItems = func() (items []messages.CompletionItem) {
		for _, version := range manifest.VersionNames() {
			items = append(items, messages.CompletionItem{
				Label: version,
				Kind:  messages.ValueCompletion,
			})
		}
		return items
	}()
)

type mockObjectsRepo struct {
	projectNames []string
}

func (m mockObjectsRepo) GetAllNames(_ context.Context, kind manifest.Kind, _ string) []string {
	if kind == manifest.KindProject {
		return m.projectNames
	}
	return nil
}

func (m mockObjectsRepo) GetDefaultProject() string {
	return "default"
}
