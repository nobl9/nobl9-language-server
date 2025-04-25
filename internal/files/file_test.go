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
	file := &File{URI: "file1"}
	file.Update(context.Background(), 1, "content")

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
		version int
		inFile  *File
		outFile *File
	}{
		{
			name:    "initial first version",
			content: "content: foo",
			version: 1,
			inFile:  &File{},
			outFile: &File{Version: 1, Content: "content: foo"},
		},
		{
			name:    "version upgrade",
			content: "content: foo",
			version: 2,
			inFile:  &File{Version: 1, Content: "content: bar"},
			outFile: &File{Version: 2, Content: "content: foo"},
		},
		{
			name:    "version upgrade from zero",
			content: "content: foo",
			version: 1,
			inFile:  &File{Version: 0, Content: "content: bar"},
			outFile: &File{Version: 1, Content: "content: foo"},
		},
		{
			name:    "version downgrade (zero)",
			content: "content: foo",
			version: 0,
			inFile:  &File{Version: 1, Content: "content: bar"},
			outFile: &File{Version: 1, Content: "content: bar"},
		},
		{
			name:    "version downgrade (non-zero)",
			content: "content: foo",
			version: 1,
			inFile:  &File{Version: 2, Content: "content: bar"},
			outFile: &File{Version: 2, Content: "content: bar"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.inFile.Update(context.Background(), tc.version, tc.content)

			assert.Equal(t, tc.outFile.Content, tc.inFile.Content)
			assert.Equal(t, tc.outFile.Version, tc.inFile.Version)
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
  labels:
    team:
    - green
    - orange
    - gray
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
							{Path: "$.metadata.labels"},
							{Path: "$.metadata.labels.team"},
							{Path: "$.metadata.labels.team[0]"},
							{Path: "$.metadata.labels.team[1]"},
							{Path: "$.metadata.labels.team[2]"},
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
					Version:       manifest.VersionV1alpha,
					isListElement: true,
					Doc: &yamlastsimple.Document{
						Offset: 0,
						Lines: []*yamlastsimple.Line{
							{Path: "$[0].apiVersion"},
						},
					},
				},
				{
					Kind:          manifest.KindService,
					isListElement: true,
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
    name: my-project-1
  spec:
    description: My project
- kind: Service
  apiVersion: n9/v1alpha
  metadata:
    name: my-service-2
    project: default
  spec:
    description: My service
- kind: Service
  apiVersion: n9/v1alpha
  metadata:
    name: my-service-3
    project: default
    labels:
      team:
      - green
      - gray
      - orange
  spec:
    description: My service`,
			expectedAST: SimpleObjectFile{
				{
					Version:       manifest.VersionV1alpha,
					Kind:          manifest.KindProject,
					isListElement: true,
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
					Version:       manifest.VersionV1alpha,
					Kind:          manifest.KindService,
					isListElement: true,
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
				{
					Version:       manifest.VersionV1alpha,
					Kind:          manifest.KindService,
					isListElement: true,
					Doc: &yamlastsimple.Document{
						Offset: 13,
						Lines: []*yamlastsimple.Line{
							{Path: "$[2].kind"},
							{Path: "$[2].apiVersion"},
							{Path: "$[2].metadata"},
							{Path: "$[2].metadata.name"},
							{Path: "$[2].metadata.project"},
							{Path: "$[2].metadata.labels"},
							{Path: "$[2].metadata.labels.team"},
							{Path: "$[2].metadata.labels.team[0]"},
							{Path: "$[2].metadata.labels.team[1]"},
							{Path: "$[2].metadata.labels.team[2]"},
							{Path: "$[2].spec"},
							{Path: "$[2].spec.description"},
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
					Version:       manifest.VersionV1alpha,
					Kind:          manifest.KindProject,
					isListElement: true,
					Doc: &yamlastsimple.Document{
						Offset: 0,
						Lines: []*yamlastsimple.Line{
							{Path: "$[0].apiVersion"},
							{Path: "$[0].kind"},
						},
					},
				},
				{
					Version:       manifest.VersionV1alpha,
					Kind:          manifest.KindService,
					isListElement: true,
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
					Kind:          manifest.KindProject,
					isListElement: true,
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
				assert.Equal(t, tc.expectedAST[i].isListElement, node.isListElement)
				require.Len(t, node.Doc.Lines, len(tc.expectedAST[i].Doc.Lines))
				for j, line := range node.Doc.Lines {
					assert.Equal(t, tc.expectedAST[i].Doc.Lines[j].Path, line.Path, "line %d", j)
				}
			}
		})
	}
}

func TestSimpleObjectNode_FindLineByPath(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		path     string
		expected *yamlastsimple.Line
	}{
		{
			name: "find existing path",
			content: `metadata:
  project: default
  name: this`,
			path:     "$.metadata.name",
			expected: &yamlastsimple.Line{Path: "$.metadata.name"},
		},
		{
			name: "find non-existing path",
			content: `metadata:
  project: default
  name: this`,
			path:     "$.metadata.description",
			expected: nil,
		},
		{
			name: "find path with list index within list element",
			content: `- metadata:
    project: default
    name: this`,
			path:     "$[0].metadata.name",
			expected: &yamlastsimple.Line{Path: "$[0].metadata.name"},
		},
		{
			name: "find path without list index within list element",
			content: `- metadata:
    project: default
    name: this`,
			path:     "$.metadata.name",
			expected: &yamlastsimple.Line{Path: "$[0].metadata.name"},
		},
		{
			name: "find path with list index within document",
			content: `metadata:
  project: default
  name: this`,
			path:     "$[0].metadata.name",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseSimpleObjectFile(tt.content)
			require.NoError(t, err)
			require.Len(t, file, 1)
			result := file[0].FindLineByPath(tt.path)
			if tt.expected == nil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.expected.Path, result.Path)
		})
	}
}
