package server

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/sdkdocs"
	"github.com/nobl9/nobl9-language-server/internal/testutils"
)

func TestDiagnosticsProvider_DiagnoseFile(t *testing.T) {
	testFilesDir := filepath.Join(testutils.FindModuleRoot(), "internal", "server", "testdata", "diagnostics")
	getTestFileURI := func(name string) messages.TextDocumentIdentifier {
		return messages.TextDocumentIdentifier{URI: filepath.Join(testFilesDir, name)}
	}

	fileSystem := newFilesystem()
	registerTestFiles(t, fileSystem, testFilesDir)

	docs, err := sdkdocs.New()
	require.NoError(t, err)
	provider := newDiagnosticsProvider(docs, objectsProviderMock{})

	srv := Server{
		files:       fileSystem,
		diagnostics: provider,
	}

	tests := map[string]struct {
		item     messages.TextDocumentItem
		expected messages.PublishDiagnosticsParams
	}{
		"missing metadata name": {
			item: messages.TextDocumentItem{
				URI:     getTestFileURI("missing-name.yaml").URI,
				Version: 1,
				Text:    "foo", // Text is not actually relevant.
			},
			expected: messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("missing-name.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "metadata.name: property is required but was empty",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(sdkSource),
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
			expected: messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("empty-name.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "property is required but was empty",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(sdkSource),
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
			expected: messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("invalid-name.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message: "string must match regular expression: " +
							"'^[a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?$' (e.g. 'my-name', '123-abc')",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(sdkSource),
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
			expected: messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("alert-silence-missing-period-duration.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "one of [duration, endTime] properties must be set, none was provided",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(sdkSource),
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
			expected: messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("slo.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "Project does not exist",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(sdkSource),
						Range: messages.Range{
							Start: messages.Position{Line: 5, Character: 11},
							End:   messages.Position{Line: 5, Character: 18},
						},
					},
					{
						Message:  "Service does not exist in Project datadog",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(sdkSource),
						Range: messages.Range{
							Start: messages.Position{Line: 12, Character: 11},
							End:   messages.Position{Line: 12, Character: 21},
						},
					},
					{
						Message:  "AlertPolicy does not exist in Project datadog",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(sdkSource),
						Range: messages.Range{
							Start: messages.Position{Line: 13, Character: 18},
							End:   messages.Position{Line: 13, Character: 21},
						},
					},
					{
						Message:  "Agent does not exist in Project datadog",
						Severity: messages.DiagnosticSeverityError,
						Source:   ptr(sdkSource),
						Range: messages.Range{
							Start: messages.Position{Line: 10, Character: 4},
							End:   messages.Position{Line: 10, Character: 16},
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
			expected: messages.PublishDiagnosticsParams{
				URI:     getTestFileURI("deprecated-composite.yaml").URI,
				Version: 1,
				Diagnostics: []messages.Diagnostic{
					{
						Message:  "property is deprecated",
						Severity: messages.DiagnosticSeverityWarning,
						Source:   ptr(sdkSource),
						Range: messages.Range{
							Start: messages.Position{Line: 7, Character: 2},
							End:   messages.Position{Line: 7, Character: 11},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			params, err := srv.handleDiagnostics(context.Background(), test.item)
			require.NoError(t, err)
			require.NotNil(t, params)
			assert.Equal(t, test.expected, *params)
		})
	}
}

type objectsProviderMock struct{}

func (o objectsProviderMock) GetObject(
	_ context.Context,
	_ manifest.Kind,
	name, project string,
) (manifest.Object, error) {
	if name == "default" || project == "default" {
		return v1alpha.GenericObject{}, nil
	}
	return nil, nil
}
