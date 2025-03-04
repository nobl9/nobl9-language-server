package completion

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nobl9/nobl9-go/manifest"
	v1alphaSLO "github.com/nobl9/nobl9-go/manifest/v1alpha/slo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/nobl9repo"
	"github.com/nobl9/nobl9-language-server/internal/objectref"
	"github.com/nobl9/nobl9-language-server/internal/sdkdocs"
	"github.com/nobl9/nobl9-language-server/internal/testutils"
)

var testDir = filepath.Join(testutils.FindModuleRoot(), "internal", "completion", "testdata")

type handlerTestCase struct {
	params           messages.CompletionParams
	expected         []messages.CompletionItem
	firstSnippetItem *messages.CompletionItem
	ignoreSnippets   bool
}

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	fileSystem := files.NewFS()
	testutils.RegisterTestFiles(t, fileSystem, testDir)

	docs, err := sdkdocs.New()
	require.NoError(t, err)
	repo := mockObjectsRepo{
		names: []string{"foo", "bar"},
	}

	handler := &Handler{
		files: fileSystem,
		providers: []providerInterface{
			NewReferencesCompletionProvider(repo),
			NewKeysCompletionProvider(docs),
			NewValuesCompletionProvider(docs),
			NewSnippetsProvider(),
		},
	}

	tests := map[string]handlerTestCase{
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
			expected: getRootPathCompletionItems(0),
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
			expected:       getRootPathCompletionItems(0),
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
			expected:       getRootPathCompletionItems(0),
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
			expected: getRootPathCompletionItems(0),
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
			expected: getRootPathCompletionItems(0),
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
			expected: getRootPathCompletionItems(2),
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
			expected: getRootPathCompletionItems(0),
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
			expected: getRootPathCompletionItems(0),
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
	tests = mergeMaps(tests, getReferenceCompletionTestCases())

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

func getReferenceCompletionTestCases() map[string]handlerTestCase {
	tests := map[string]messages.Position{
		"service project names - value start":         {Line: 4, Character: 11},
		"service project names - value end":           {Line: 4, Character: 12},
		"alert policy - alert method name":            {Line: 20, Character: 19},
		"alert policy - alert method project":         {Line: 21, Character: 19},
		"alert policy - default to metadata project":  {Line: 43, Character: 19},
		"alert silence - alert policy name":           {Line: 62, Character: 17},
		"alert silence - alert policy project":        {Line: 63, Character: 17},
		"alert silence - default to metadata project": {Line: 76, Character: 14},
		"annotation - slo project":                    {Line: 86, Character: 10},
		"budget adjustment - slo name with project":   {Line: 105, Character: 19},
		"report - project":                            {Line: 117, Character: 11},
		"report - service":                            {Line: 120, Character: 18},
		"report - service project":                    {Line: 121, Character: 18},
		"report - slo":                                {Line: 124, Character: 18},
		"report - slo project":                        {Line: 125, Character: 18},
		"role binding - project ref":                  {Line: 140, Character: 20},
		"role binding - group ref":                    {Line: 146, Character: 20},
	}

	testCasesMap := make(map[string]handlerTestCase, len(tests))
	for name, tc := range tests {
		testCasesMap[name] = handlerTestCase{
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("complete-refs.yaml"),
					Position: messages.Position{
						Line:      tc.Line,
						Character: tc.Character,
					},
				},
			},
			expected: []messages.CompletionItem{
				{Label: "foo", Kind: messages.ReferenceCompletion},
				{Label: "bar", Kind: messages.ReferenceCompletion},
			},
		}
	}

	noCompletionTests := map[string]messages.Position{
		"budget adjustment - slo name with no project": {Line: 104, Character: 19},
		"report - service with no project":             {Line: 119, Character: 17},
		"report - slo with no project":                 {Line: 123, Character: 17},
	}
	for name, tc := range noCompletionTests {
		testCasesMap[name] = handlerTestCase{
			params: messages.CompletionParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("complete-refs.yaml"),
					Position: messages.Position{
						Line:      tc.Line,
						Character: tc.Character,
					},
				},
			},
		}
	}

	// Custom.
	testCasesMap["role binding - user ref"] = handlerTestCase{
		params: messages.CompletionParams{
			TextDocumentPositionParams: messages.TextDocumentPositionParams{
				TextDocument: getTestFileURI("complete-refs.yaml"),
				Position: messages.Position{
					Line:      138,
					Character: 16,
				},
			},
		},
		expected: []messages.CompletionItem{
			{
				Label:            "Foo Bar (default@baz.com)",
				Kind:             messages.ReferenceCompletion,
				InsertText:       "foo",
				InsertTextFormat: messages.PlainTextTextFormat,
			},
		},
	}
	testCasesMap["role binding - project role ref"] = handlerTestCase{
		params: messages.CompletionParams{
			TextDocumentPositionParams: messages.TextDocumentPositionParams{
				TextDocument: getTestFileURI("complete-refs.yaml"),
				Position: messages.Position{
					Line:      139,
					Character: 20,
				},
			},
		},
		expected: []messages.CompletionItem{
			{Label: "project-viewer", Kind: messages.ReferenceCompletion},
		},
	}
	testCasesMap["role binding - org role ref"] = handlerTestCase{
		params: messages.CompletionParams{
			TextDocumentPositionParams: messages.TextDocumentPositionParams{
				TextDocument: getTestFileURI("complete-refs.yaml"),
				Position: messages.Position{
					Line:      147,
					Character: 17,
				},
			},
		},
		expected: []messages.CompletionItem{
			{Label: "organization-admin", Kind: messages.ReferenceCompletion},
		},
	}
	return testCasesMap
}

func getTestFileURI(name string) messages.TextDocumentIdentifier {
	return messages.TextDocumentIdentifier{URI: filepath.Join(testDir, name)}
}

func getRootPathCompletionItems(indent int) []messages.CompletionItem {
	return []messages.CompletionItem{
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
			InsertText:       "metadata:\n  " + strings.Repeat(" ", indent),
			InsertTextFormat: messages.PlainTextTextFormat,
		},
		{
			Label:            "spec",
			Kind:             messages.PropertyCompletion,
			InsertText:       "spec:\n  " + strings.Repeat(" ", indent),
			InsertTextFormat: messages.PlainTextTextFormat,
		},
	}
}

var (
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
	names []string
}

func (m mockObjectsRepo) GetAllNames(_ context.Context, kind manifest.Kind, project string) []string {
	if !objectref.IsProjectScoped(kind) {
		return m.names
	}
	if project == "" {
		panic("project must be set for kind " + kind.String())
	}
	return m.names
}

func (m mockObjectsRepo) GetObject(
	_ context.Context,
	kind manifest.Kind,
	_, _ string,
) (manifest.Object, error) {
	if kind == manifest.KindSLO {
		return v1alphaSLO.New(
			v1alphaSLO.Metadata{},
			v1alphaSLO.Spec{
				Objectives: []v1alphaSLO.Objective{
					{ObjectiveBase: v1alphaSLO.ObjectiveBase{Name: "foo"}},
					{ObjectiveBase: v1alphaSLO.ObjectiveBase{Name: "bar"}},
				},
			},
		), nil
	}
	return nil, nil
}

func (m mockObjectsRepo) GetDefaultProject() string {
	return "default"
}

func (m mockObjectsRepo) GetUsers(_ context.Context, phrase string) ([]*nobl9repo.User, error) {
	return []*nobl9repo.User{
		{UserID: "foo", FirstName: "Foo", LastName: "Bar", Email: phrase + "@baz.com"},
	}, nil
}

func (m mockObjectsRepo) GetRoles(_ context.Context) (*nobl9repo.Roles, error) {
	return &nobl9repo.Roles{
		OrganizationRoles: []nobl9repo.Role{{Name: "organization-admin"}},
		ProjectRoles:      []nobl9repo.Role{{Name: "project-viewer"}},
	}, nil
}

func mergeMaps[V any](maps ...map[string]V) map[string]V {
	result := make(map[string]V)
	for _, m := range maps {
		for k, v := range m {
			if _, ok := result[k]; ok {
				panic("duplicate key: " + k)
			}
			result[k] = v
		}
	}
	return result
}
