package diagnostics

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha"
	v1alphaSLO "github.com/nobl9/nobl9-go/manifest/v1alpha/slo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-language-server/internal/config"
	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/nobl9repo"
	"github.com/nobl9/nobl9-language-server/internal/sdkdocs"
	"github.com/nobl9/nobl9-language-server/internal/testutils"
)

func TestHandler_Handle(t *testing.T) {
	testFilesDir := filepath.Join(testutils.FindModuleRoot(), "internal", "diagnostics", "testdata")
	getTestFileURI := func(name string) messages.TextDocumentIdentifier {
		return messages.TextDocumentIdentifier{URI: filepath.Join(testFilesDir, name)}
	}

	fileSystem := files.NewFS(nil)
	testutils.RegisterTestFiles(t, fileSystem, testFilesDir)

	docs, err := sdkdocs.New()
	require.NoError(t, err)
	provider := NewProvider(docs, objectsProviderMock{})

	handler := Handler{
		fs:          fileSystem,
		diagnostics: provider,
	}

	tests := map[string]struct {
		item     messages.TextDocumentItem
		expected *messages.PublishDiagnosticsParams
	}{
		"missing metadata name": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("missing-name.yaml").URI,
				Version: 1,
				Text:    "foo", // Text is not actually relevant.
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("missing-name.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "metadata.name: property is required but was empty",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 2, Character: 0},
							End:   messages.Position{Line: 2, Character: 8},
						},
					},
				},
			},
		},
		"missing metadata name - no TextDocumentItem.Text": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("missing-name.yaml").URI,
				Version: 1,
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("missing-name.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "metadata.name: property is required but was empty",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 2, Character: 0},
							End:   messages.Position{Line: 2, Character: 8},
						},
					},
				},
			},
		},
		"empty metadata name": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("empty-name.yaml").URI,
				Version: 1,
				Text:    "foo",
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("empty-name.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "property is required but was empty",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 3, Character: 2},
							End:   messages.Position{Line: 3, Character: 6},
						},
					},
				},
			},
		},
		"invalid metadata name": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("invalid-name.yaml").URI,
				Version: 1,
				Text:    "foo",
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("invalid-name.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message: "string must match regular expression: " +
							"'^[a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?$' (e.g. 'my-name', '123-abc')" +
							"; an RFC-1123 compliant label name must consist of lower case alphanumeric characters" +
							" or '-', and must start and end with an alphanumeric character",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 4, Character: 8},
							End:   messages.Position{Line: 4, Character: 14},
						},
					},
				},
			},
		},
		"alert silence missing period duration": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("alert-silence-missing-period-duration.yaml").URI,
				Version: 1,
				Text:    "foo",
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("alert-silence-missing-period-duration.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "one of [duration, endTime] properties must be set, none was provided",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 8, Character: 2},
							End:   messages.Position{Line: 8, Character: 8},
						},
					},
				},
			},
		},
		"slo referenced objects do not exist": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("slo.yaml").URI,
				Version: 1,
				Text:    "foo",
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("slo.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "Agent does not exist in Project default",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 11, Character: 12},
							End:   messages.Position{Line: 11, Character: 19},
						},
					},
					{
						Message:  "Service does not exist in Project default",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 12, Character: 11},
							End:   messages.Position{Line: 12, Character: 21},
						},
					},
					{
						Message:  "AlertPolicy does not exist in Project default",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 13, Character: 18},
							End:   messages.Position{Line: 13, Character: 21},
						},
					},
					{
						Message:  "Project does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 60, Character: 23},
							End:   messages.Position{Line: 60, Character: 26},
						},
					},
					{
						Message:  "Project does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 77, Character: 19},
							End:   messages.Position{Line: 77, Character: 22},
						},
					},
					{
						Message:  "SLO does not exist in Project default",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 102, Character: 19},
							End:   messages.Position{Line: 102, Character: 22},
						},
					},
					{
						Message:  "AlertMethod does not exist in Project default",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 116, Character: 16},
							End:   messages.Position{Line: 116, Character: 34},
						},
					},
					{
						Message:  "objective does not exist in SLO default and Project default",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 143, Character: 25},
							End:   messages.Position{Line: 143, Character: 28},
						},
					},
					{
						Message:  "Project does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 194, Character: 11},
							End:   messages.Position{Line: 194, Character: 18},
						},
					},
				},
			},
		},
		"service referenced objects do not exist": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("service.yaml").URI,
				Version: 1,
				Text:    "foo",
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("service.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "Project does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 5, Character: 11},
							End:   messages.Position{Line: 5, Character: 17},
						},
					},
				},
			},
		},
		"alert policy referenced objects do not exist": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("alert-policy.yaml").URI,
				Version: 1,
				Text:    "foo",
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("alert-policy.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "Project does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 5, Character: 13},
							End:   messages.Position{Line: 5, Character: 19},
						},
					},
					{
						Message:  "AlertMethod does not exist in Project default",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 34, Character: 16},
							End:   messages.Position{Line: 34, Character: 21},
						},
					},
					{
						Message:  "Project does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 79, Character: 19},
							End:   messages.Position{Line: 79, Character: 25},
						},
					},
				},
			},
		},
		"alert silence referenced objects do not exist": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("alert-silence.yaml").URI,
				Version: 1,
				Text:    "foo",
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("alert-silence.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "Project does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 4, Character: 11},
							End:   messages.Position{Line: 4, Character: 17},
						},
					},
					{
						Message:  "SLO does not exist in Project default",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 20, Character: 7},
							End:   messages.Position{Line: 20, Character: 36},
						},
					},
					{
						Message:  "AlertPolicy does not exist in Project default",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 23, Character: 10},
							End:   messages.Position{Line: 23, Character: 22},
						},
					},
				},
			},
		},
		"annotation referenced objects do not exist": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("annotation.yaml").URI,
				Version: 1,
				Text:    "foo",
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("annotation.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "Project does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 4, Character: 11},
							End:   messages.Position{Line: 4, Character: 17},
						},
					},
					{
						Message:  "SLO does not exist in Project default",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 17, Character: 7},
							End:   messages.Position{Line: 17, Character: 36},
						},
					},
					{
						Message:  "objective does not exist in SLO default and Project default",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 29, Character: 17},
							End:   messages.Position{Line: 29, Character: 23},
						},
					},
				},
			},
		},
		"budget adjustment referenced objects do not exist": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("budget-adjustment.yaml").URI,
				Version: 1,
				Text:    "foo",
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("budget-adjustment.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "Project does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 15, Character: 17},
							End:   messages.Position{Line: 15, Character: 23},
						},
					},
					{
						Message:  "SLO does not exist in Project default",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 16, Character: 14},
							End:   messages.Position{Line: 16, Character: 31},
						},
					},
				},
			},
		},
		"report referenced objects do not exist": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("report.yaml").URI,
				Version: 1,
				Text:    "foo",
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("report.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "Project does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 10, Character: 10},
							End:   messages.Position{Line: 10, Character: 16},
						},
					},
					{
						Message:  "Project does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 27, Character: 19},
							End:   messages.Position{Line: 27, Character: 25},
						},
					},
					{
						Message:  "Service does not exist in Project default",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 28, Character: 16},
							End:   messages.Position{Line: 28, Character: 25},
						},
					},
					{
						Message:  "Project does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 34, Character: 19},
							End:   messages.Position{Line: 34, Character: 25},
						},
					},
					{
						Message:  "SLO does not exist in Project default",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 35, Character: 16},
							End:   messages.Position{Line: 35, Character: 21},
						},
					},
				},
			},
		},
		"role binding referenced objects do not exist": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("role-binding.yaml").URI,
				Version: 1,
				Text:    "foo",
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("role-binding.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "user does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 5, Character: 10},
							End:   messages.Position{Line: 5, Character: 30},
						},
					},
					{
						Message:  "organization role does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 6, Character: 13},
							End:   messages.Position{Line: 6, Character: 31},
						},
					},
					{
						Message:  "UserGroup does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 19, Character: 14},
							End:   messages.Position{Line: 19, Character: 32},
						},
					},
					{
						Message:  "project role does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 34, Character: 13},
							End:   messages.Position{Line: 34, Character: 27},
						},
					},
					{
						Message:  "project role does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 50, Character: 13},
							End:   messages.Position{Line: 50, Character: 27},
						},
					},
				},
			},
		},
		"user group referenced objects do not exist": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("user-group.yaml").URI,
				Version: 1,
				Text:    "foo",
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("user-group.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "user does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 7, Character: 10},
							End:   messages.Position{Line: 7, Character: 17},
						},
					},
				},
			},
		},
		"deprecated composite": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("deprecated-composite.yaml").URI,
				Version: 1,
				Text:    "foo",
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("deprecated-composite.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "property is deprecated",
						Severity: messages.DiagnosticSeverityWarning,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 7, Character: 2},
							End:   messages.Position{Line: 7, Character: 11},
						},
					},
					{
						Message:  "property is deprecated",
						Severity: messages.DiagnosticSeverityWarning,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 52, Character: 4},
							End:   messages.Position{Line: 52, Character: 13},
						},
					},
				},
			},
		},
		"invalid composite": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("invalid-composite.yaml").URI,
				Version: 1,
				Text:    "foo",
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("invalid-composite.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "property is forbidden; indicator section is forbidden when spec.objectives[0].composite is provided",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 19, Character: 2},
							End:   messages.Position{Line: 19, Character: 11},
						},
					},
					{
						Message:  "spec.objectives[0].composite.components.objectives[0].weight: should be greater than '0'",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 34, Character: 12},
							End:   messages.Position{Line: 34, Character: 22},
						},
					},
					{
						Message:  "spec.objectives[0].composite.components.objectives[0].whenDelayed: property is required but was empty",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(config.ServerName),
						Range: messages.Range{
							Start: messages.Position{Line: 34, Character: 12},
							End:   messages.Position{Line: 34, Character: 22},
						},
					},
				},
			},
		},
		"barebones object": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("barebones-object.yaml").URI,
				Version: 1,
				Text:    "foo", // Text is not actually relevant.
			},
			expected: &messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("barebones-object.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "non-map value is specified",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(goYamlSource),
						Range: messages.Range{
							Start: messages.Position{Line: 1, Character: 1},
							End:   messages.Position{Line: 1, Character: 1},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			params, err := handler.Handle(context.Background(), test.item)
			require.NoError(t, err)
			require.NotNil(t, params)
			assert.Equal(t, test.expected, params)
		})
	}
}

type objectsProviderMock struct{}

func (o objectsProviderMock) GetObject(
	_ context.Context,
	kind manifest.Kind,
	name, project string,
) (manifest.Object, error) {
	if name == "default" || (name == "" && project == "default") {
		if kind == manifest.KindSLO {
			return v1alphaSLO.New(
				v1alphaSLO.Metadata{},
				v1alphaSLO.Spec{
					Objectives: []v1alphaSLO.Objective{
						{ObjectiveBase: v1alphaSLO.ObjectiveBase{Name: "default"}},
					},
				},
			), nil
		}
		return v1alpha.GenericObject{}, nil
	}
	return nil, nil
}

func (o objectsProviderMock) GetDefaultProject() string {
	return "default"
}

func (o objectsProviderMock) GetUser(_ context.Context, id string) (*nobl9repo.User, error) {
	if id == "default" {
		return &nobl9repo.User{}, nil
	}
	return nil, nil
}

func (o objectsProviderMock) GetRoles(_ context.Context) (*nobl9repo.Roles, error) {
	return &nobl9repo.Roles{
		OrganizationRoles: []nobl9repo.Role{{Name: "default"}},
		ProjectRoles:      []nobl9repo.Role{{Name: "default"}},
	}, nil
}
