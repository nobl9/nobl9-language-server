package files

import (
	"context"
	"testing"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-language-server/internal/yamlastsimple"
)

func TestNewFile(t *testing.T) {
	file := NewFile(context.Background(), "file1", 1, "content")

	require.NotNil(t, file)
	assert.Equal(t, "file1", file.URI)
	assert.Equal(t, "content", file.Content)
	assert.Equal(t, 1, file.Version)
	assert.NoError(t, file.Err)
}

func TestFile_Update(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "invalid Nobl9 object",
			content: "content: foo",
		},
		{
			name:    "invalid YAML",
			content: "content:foo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file := &File{Version: 1}
			file.Update(context.Background(), 2, tc.content)

			assert.Equal(t, tc.content, file.Content)
			assert.Equal(t, 2, file.Version)
		})
	}
}

func TestFile_Update_SimpleAST(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectedAST SimpleObjectFile
	}{
		{
			name:    "invalid Nobl9 object",
			content: "content: foo",
			expectedAST: SimpleObjectFile{
				{
					Doc: &yamlastsimple.Document{
						Offset: 0,
						Lines: []*yamlastsimple.Line{
							{Path: "$.content"},
						},
					},
				},
			},
		},
		{
			name:    "invalid YAML",
			content: "content:foo",
			expectedAST: SimpleObjectFile{
				{
					Doc: &yamlastsimple.Document{
						Offset: 0,
						Lines: []*yamlastsimple.Line{
							{Path: "$.content"},
						},
					},
				},
			},
		},
		{
			name:    "only version",
			content: "apiVersion: n9/v1alpha",
			expectedAST: SimpleObjectFile{
				{
					Version: manifest.VersionV1alpha,
					Doc: &yamlastsimple.Document{
						Offset: 0,
						Lines: []*yamlastsimple.Line{
							{Path: "$.apiVersion"},
						},
					},
				},
			},
		},
		{
			name:    "invalid version",
			content: "apiVersion: nobl9.eu/v1alpha",
			expectedAST: SimpleObjectFile{
				{
					Doc: &yamlastsimple.Document{
						Offset: 0,
						Lines: []*yamlastsimple.Line{
							{Path: "$.apiVersion"},
						},
					},
				},
			},
		},
		{
			name:    "only kind",
			content: "kind: Service",
			expectedAST: SimpleObjectFile{
				{
					Kind: manifest.KindService,
					Doc: &yamlastsimple.Document{
						Offset: 0,
						Lines: []*yamlastsimple.Line{
							{Path: "$.kind"},
						},
					},
				},
			},
		},
		{
			name:    "invalid kind",
			content: "kind: Foo",
			expectedAST: SimpleObjectFile{
				{
					Doc: &yamlastsimple.Document{
						Offset: 0,
						Lines: []*yamlastsimple.Line{
							{Path: "$.kind"},
						},
					},
				},
			},
		},
		{
			name: "service object",
			content: `apiVersion: n9/v1alpha
kind: Service
metadata:
  name: my-service
  project: default
spec:
  description: My service`,
			expectedAST: SimpleObjectFile{
				{
					Version: manifest.VersionV1alpha,
					Kind:    manifest.KindService,
					Doc: &yamlastsimple.Document{
						Offset: 0,
						Lines: []*yamlastsimple.Line{
							{Path: "$.apiVersion"},
							{Path: "$.kind"},
							{Path: "$.metadata"},
							{Path: "$.metadata.name"},
							{Path: "$.metadata.project"},
							{Path: "$.spec"},
							{Path: "$.spec.description"},
						},
					},
				},
			},
		},
		{
			name: "documents",
			content: `apiVersion: n9/v1alpha
---
kind: Service`,
			expectedAST: SimpleObjectFile{
				{
					Version: manifest.VersionV1alpha,
					Doc: &yamlastsimple.Document{
						Offset: 0,
						Lines: []*yamlastsimple.Line{
							{Path: "$.apiVersion"},
							{Path: ""},
						},
					},
				},
				{
					Kind: manifest.KindService,
					Doc: &yamlastsimple.Document{
						Offset: 2,
						Lines: []*yamlastsimple.Line{
							{Path: "$.kind"},
						},
					},
				},
			},
		},
		{
			name: "list of incomplete objects",
			content: `- apiVersion: n9/v1alpha
- kind: Service`,
			expectedAST: SimpleObjectFile{
				{
					Version: manifest.VersionV1alpha,
					Doc: &yamlastsimple.Document{
						Offset: 0,
						Lines: []*yamlastsimple.Line{
							{Path: "$[0].apiVersion"},
						},
					},
				},
				{
					Kind: manifest.KindService,
					Doc: &yamlastsimple.Document{
						Offset: 1,
						Lines: []*yamlastsimple.Line{
							{Path: "$[1].kind"},
						},
					},
				},
			},
		},
		{
			name: "list of objects",
			content: `- apiVersion: n9/v1alpha
  kind: Project
  metadata:
    name: my-project
  spec:
    description: My project
- kind: Service
  apiVersion: n9/v1alpha
  metadata:
    name: my-service
    project: default
  spec:
    description: My service`,
			expectedAST: SimpleObjectFile{
				{
					Version: manifest.VersionV1alpha,
					Kind:    manifest.KindProject,
					Doc: &yamlastsimple.Document{
						Offset: 0,
						Lines: []*yamlastsimple.Line{
							{Path: "$[0].apiVersion"},
							{Path: "$[0].kind"},
							{Path: "$[0].metadata"},
							{Path: "$[0].metadata.name"},
							{Path: "$[0].spec"},
							{Path: "$[0].spec.description"},
						},
					},
				},
				{
					Version: manifest.VersionV1alpha,
					Kind:    manifest.KindService,
					Doc: &yamlastsimple.Document{
						Offset: 6,
						Lines: []*yamlastsimple.Line{
							{Path: "$[1].kind"},
							{Path: "$[1].apiVersion"},
							{Path: "$[1].metadata"},
							{Path: "$[1].metadata.name"},
							{Path: "$[1].metadata.project"},
							{Path: "$[1].spec"},
							{Path: "$[1].spec.description"},
						},
					},
				},
			},
		},
		{
			name: "mix of documents",
			content: `- apiVersion: n9/v1alpha
  kind: Project
- kind: Service
  apiVersion: n9/v1alpha
---
apiVersion: n9/v1alpha
---
- kind: Project
`,
			expectedAST: SimpleObjectFile{
				{
					Version: manifest.VersionV1alpha,
					Kind:    manifest.KindProject,
					Doc: &yamlastsimple.Document{
						Offset: 0,
						Lines: []*yamlastsimple.Line{
							{Path: "$[0].apiVersion"},
							{Path: "$[0].kind"},
						},
					},
				},
				{
					Version: manifest.VersionV1alpha,
					Kind:    manifest.KindService,
					Doc: &yamlastsimple.Document{
						Offset: 2,
						Lines: []*yamlastsimple.Line{
							{Path: "$[1].kind"},
							{Path: "$[1].apiVersion"},
							{Path: ""},
						},
					},
				},
				{
					Version: manifest.VersionV1alpha,
					Doc: &yamlastsimple.Document{
						Offset: 5,
						Lines: []*yamlastsimple.Line{
							{Path: "$.apiVersion"},
							{Path: ""},
						},
					},
				},
				{
					Kind: manifest.KindProject,
					Doc: &yamlastsimple.Document{
						Offset: 7,
						Lines: []*yamlastsimple.Line{
							{Path: "$[0].kind"},
							{Path: "$"},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file := &File{Version: 1}
			file.Update(context.Background(), 2, tc.content)

			require.Len(t, file.SimpleAST, len(tc.expectedAST))
			for i, node := range file.SimpleAST {
				assert.Equal(t, tc.expectedAST[i].Version, node.Version)
				assert.Equal(t, tc.expectedAST[i].Kind, node.Kind)
				assert.Equal(t, tc.expectedAST[i].Doc.Offset, node.Doc.Offset)
				require.Len(t, node.Doc.Lines, len(tc.expectedAST[i].Doc.Lines))
				for j, line := range node.Doc.Lines {
					assert.Equal(t, tc.expectedAST[i].Doc.Lines[j].Path, line.Path, "line %d", j)
				}
			}
		})
	}
}

func TestFile_Update_NoVersionChange(t *testing.T) {
	file := &File{Version: 1, Content: "old content"}
	file.Update(context.Background(), 1, "new content")

	assert.Equal(t, "old content", file.Content)
	assert.Equal(t, 1, file.Version)
}

func TestFile_Update_ZeroVersion(t *testing.T) {
	file := &File{Version: 1, Content: "old content"}
	file.Update(context.Background(), 0, "new content")

	assert.Equal(t, "new content", file.Content)
	assert.Equal(t, 1, file.Version)
}
