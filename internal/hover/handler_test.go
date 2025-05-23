package hover

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha"
	v1alphaAgent "github.com/nobl9/nobl9-go/manifest/v1alpha/agent"
	v1alphaProject "github.com/nobl9/nobl9-go/manifest/v1alpha/project"
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

var (
	testDir    = filepath.Join(testutils.FindModuleRoot(), "internal", "hover", "testdata")
	outputsDir = filepath.Join(testDir, "outputs")
	inputsDir  = filepath.Join(testDir, "inputs")
)

type handlerTestCase struct {
	params   messages.HoverParams
	expected *messages.HoverResponse
}

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	fileSystem := files.NewFS(nil)
	testutils.RegisterTestFiles(t, fileSystem, inputsDir)

	docs, err := sdkdocs.New()
	require.NoError(t, err)
	repo := mockObjectsRepo{
		names: []string{"foo", "bar"},
	}

	handler := &Handler{
		files:    fileSystem,
		provider: NewProvider(docs, repo),
	}

	tests := map[string]handlerTestCase{
		"service - apiVersion key": {
			params: messages.HoverParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("service.yaml"),
					Position: messages.Position{
						Line:      0,
						Character: 9,
					},
				},
			},
			expected: &messages.HoverResponse{
				Contents: messages.MarkupContent{
					Kind:  messages.Markdown,
					Value: mustReadFile(t, "apiversion.md"),
				},
			},
		},
		"service - apiVersion value": {
			params: messages.HoverParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("service.yaml"),
					Position: messages.Position{
						Line:      0,
						Character: 18,
					},
				},
			},
			expected: &messages.HoverResponse{
				Contents: messages.MarkupContent{
					Kind:  messages.Markdown,
					Value: mustReadFile(t, "apiversion.md"),
				},
			},
		},
		"service - metadata.project key": {
			params: messages.HoverParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("service.yaml"),
					Position: messages.Position{
						Line:      4,
						Character: 6,
					},
				},
			},
			expected: &messages.HoverResponse{
				Contents: messages.MarkupContent{
					Kind:  messages.Markdown,
					Value: mustReadFile(t, "metadata-project-key.md"),
				},
			},
		},
		"service - metadata.project value": {
			params: messages.HoverParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("service.yaml"),
					Position: messages.Position{
						Line:      4,
						Character: 16,
					},
				},
			},
			expected: &messages.HoverResponse{
				Contents: messages.MarkupContent{
					Kind:  messages.Markdown,
					Value: mustReadFile(t, "metadata-project-value.md"),
				},
			},
		},
		"agent - metadata.project value": {
			params: messages.HoverParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("agent.yaml"),
					Position: messages.Position{
						Line:      4,
						Character: 13,
					},
				},
			},
			expected: &messages.HoverResponse{
				Contents: messages.MarkupContent{
					Kind:  messages.Markdown,
					Value: mustReadFile(t, "metadata-project-value-no-description.md"),
				},
			},
		},
		"service - metadata.project value with non-existent project": {
			params: messages.HoverParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("service.yaml"),
					Position: messages.Position{
						Line:      12,
						Character: 16,
					},
				},
			},
			expected: &messages.HoverResponse{
				Contents: messages.MarkupContent{
					Kind:  messages.Markdown,
					Value: mustReadFile(t, "metadata-project-key.md"),
				},
			},
		},
		"slo - metric source value": {
			params: messages.HoverParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("slo.yaml"),
					Position: messages.Position{
						Line:      11,
						Character: 18,
					},
				},
			},
			expected: &messages.HoverResponse{
				Contents: messages.MarkupContent{
					Kind:  messages.Markdown,
					Value: mustReadFile(t, "metric-source-value.md"),
				},
			},
		},
		"role binding - user value": {
			params: messages.HoverParams{
				TextDocumentPositionParams: messages.TextDocumentPositionParams{
					TextDocument: getTestFileURI("role-binding.yaml"),
					Position: messages.Position{
						Line:      5,
						Character: 16,
					},
				},
			},
			expected: &messages.HoverResponse{
				Contents: messages.MarkupContent{
					Kind:  messages.Markdown,
					Value: mustReadFile(t, "user-details.md"),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := handler.Handle(context.Background(), test.params)
			require.NoError(t, err)
			response := result.(*messages.HoverResponse)
			if test.expected != nil {
				require.NotNil(t, response)
			}
			assert.Equal(t, test.expected, response)
		})
	}
}

func mustReadFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(outputsDir, name))
	require.NoError(t, err)
	return string(data)
}

func getTestFileURI(name string) messages.TextDocumentIdentifier {
	return messages.TextDocumentIdentifier{URI: filepath.Join(inputsDir, name)}
}

type mockObjectsRepo struct {
	names []string
}

func (m mockObjectsRepo) GetAllNames(_ context.Context, kind manifest.Kind, project string) ([]string, error) {
	if !objectref.IsProjectScoped(kind) {
		return m.names, nil
	}
	if project == "" {
		panic("project must be set for kind " + kind.String())
	}
	return m.names, nil
}

func (m mockObjectsRepo) GetObject(
	_ context.Context,
	kind manifest.Kind,
	name, _ string,
) (manifest.Object, error) {
	switch kind {
	case manifest.KindSLO:
		if name != "default" {
			return nil, nil
		}
		return v1alphaSLO.New(
			v1alphaSLO.Metadata{},
			v1alphaSLO.Spec{
				Objectives: []v1alphaSLO.Objective{
					{ObjectiveBase: v1alphaSLO.ObjectiveBase{Name: "foo"}},
					{ObjectiveBase: v1alphaSLO.ObjectiveBase{Name: "bar"}},
				},
			},
		), nil
	case manifest.KindAgent:
		if name != "default" {
			return nil, nil
		}
		return v1alphaAgent.New(
			v1alphaAgent.Metadata{
				Name:    "default",
				Project: "default",
			},
			v1alphaAgent.Spec{
				Description:    "This is an example Agent!",
				GCM:            &v1alphaAgent.GCMConfig{},
				ReleaseChannel: v1alpha.ReleaseChannelStable,
			},
		), nil
	case manifest.KindProject:
		switch name {
		case "default":
			return v1alphaProject.New(
				v1alphaProject.Metadata{
					Name: "default",
				},
				v1alphaProject.Spec{
					Description: "This is an example Project!",
				},
			), nil
		case "no-description":
			return v1alphaProject.New(
				v1alphaProject.Metadata{
					Name: "no-description",
				},
				v1alphaProject.Spec{},
			), nil
		}
		return nil, nil
	default:
		return nil, nil
	}
}

func (m mockObjectsRepo) GetDefaultProject() string {
	return "default"
}

func (m mockObjectsRepo) GetUser(_ context.Context, id string) (*nobl9repo.User, error) {
	return &nobl9repo.User{
		UserID:    "foo",
		FirstName: "Foo",
		LastName:  "Bar",
		Email:     id + "@baz.com",
	}, nil
}
