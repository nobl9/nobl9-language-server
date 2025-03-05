package yamlastsimple

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nolint: lll
func TestParseFile(t *testing.T) {
	tests := map[string]struct {
		in  string
		out File
	}{
		"simple": {
			in: `
metadata:
  name: my-service
  annotations:
    app: my-app`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$", indent: 0, Type: LineTypeEmpty},
							{Path: "$.metadata", indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.name", indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.annotations", indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.annotations.app", indent: 4, Type: LineTypeMapping},
						},
					},
				},
			},
		},
		"empty mapping value": {
			in: `metadata:
app: my-app`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$.metadata", indent: 0, Type: LineTypeMapping},
							{Path: "$.app", indent: 0, Type: LineTypeMapping},
						},
					},
				},
			},
		},
		"comments": {
			in: `
metadata:
  name:
#annotations:
    app: my-app`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$", indent: 0, Type: LineTypeEmpty},
							{Path: "$.metadata", indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.name", indent: 2, Type: LineTypeMapping},
							{Path: "", indent: 0, Type: LineTypeComment},
							{Path: "$.metadata.name.app", indent: 4, Type: LineTypeMapping},
						},
					},
				},
			},
		},
		"empty lines": {
			in: `
metadata:
  name:

  
    app: my-app`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$", indent: 0, Type: LineTypeEmpty},
							{Path: "$.metadata", indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.name", indent: 2, Type: LineTypeMapping},
							{Path: "$", indent: 0, Type: LineTypeEmpty},
							{Path: "$.metadata", indent: 2, Type: LineTypeEmpty},
							{Path: "$.metadata.name.app", indent: 4, Type: LineTypeMapping},
						},
					},
				},
			},
		},
		"last line empty": {
			in: `
metadata:
  name: my-service
`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$", indent: 0, Type: LineTypeEmpty},
							{Path: "$.metadata", indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.name", indent: 2, Type: LineTypeMapping},
							{Path: "$", indent: 0, Type: LineTypeEmpty},
						},
					},
				},
			},
		},
		"only mappings": {
			in: `
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    app: my-app
  displayName: My Service
spec:
  description: My service`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$", indent: 0, Type: LineTypeEmpty},
							{Path: "$.apiVersion", indent: 0, Type: LineTypeMapping},
							{Path: "$.kind", indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata", indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.name", indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.annotations", indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.annotations.app", indent: 4, Type: LineTypeMapping},
							{Path: "$.metadata.displayName", indent: 2, Type: LineTypeMapping},
							{Path: "$.spec", indent: 0, Type: LineTypeMapping},
							{Path: "$.spec.description", indent: 2, Type: LineTypeMapping},
						},
					},
				},
			},
		},
		"indented list": {
			in: `
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    app: my-app
  labels:
    team:
      - dev
      - ops
  displayName: My Service
spec:
  description: My service`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$", indent: 0, Type: LineTypeEmpty},
							{Path: "$.apiVersion", indent: 0, Type: LineTypeMapping},
							{Path: "$.kind", indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata", indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.name", indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.annotations", indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.annotations.app", indent: 4, Type: LineTypeMapping},
							{Path: "$.metadata.labels", indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.labels.team", indent: 4, Type: LineTypeMapping},
							{Path: "$.metadata.labels.team[0]", GeneralizedPath: "$.metadata.labels.team[*]", indent: 8, Type: LineTypeList},
							{Path: "$.metadata.labels.team[1]", GeneralizedPath: "$.metadata.labels.team[*]", indent: 8, Type: LineTypeList},
							{Path: "$.metadata.displayName", indent: 2, Type: LineTypeMapping},
							{Path: "$.spec", indent: 0, Type: LineTypeMapping},
							{Path: "$.spec.description", indent: 2, Type: LineTypeMapping},
						},
					},
				},
			},
		},
		"inlined list": {
			in: `labels:
  team:
  - dev
  - ops
displayName: My Service`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$.labels", indent: 0, Type: LineTypeMapping},
							{Path: "$.labels.team", indent: 2, Type: LineTypeMapping},
							{Path: "$.labels.team[0]", GeneralizedPath: "$.labels.team[*]", indent: 4, Type: LineTypeList},
							{Path: "$.labels.team[1]", GeneralizedPath: "$.labels.team[*]", indent: 4, Type: LineTypeList},
							{Path: "$.displayName", indent: 0, Type: LineTypeMapping},
						},
					},
				},
			},
		},
		"list of objects": {
			in: `
metadata:
  annotations:
    - app: this
      my-app:
        foo: bar
        list:
        - dev
        - ops
        - prod
    - team: green
  displayName: my-service-1`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$", indent: 0, Type: LineTypeEmpty},
							{Path: "$.metadata", indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.annotations", indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.annotations[0].app", GeneralizedPath: "$.metadata.annotations[*].app", indent: 6, Type: LineTypeList | LineTypeMapping},
							{Path: "$.metadata.annotations[0].my-app", GeneralizedPath: "$.metadata.annotations[*].my-app", indent: 6, Type: LineTypeMapping},
							{Path: "$.metadata.annotations[0].my-app.foo", GeneralizedPath: "$.metadata.annotations[*].my-app.foo", indent: 8, Type: LineTypeMapping},
							{Path: "$.metadata.annotations[0].my-app.list", GeneralizedPath: "$.metadata.annotations[*].my-app.list", indent: 8, Type: LineTypeMapping},
							{Path: "$.metadata.annotations[0].my-app.list[0]", GeneralizedPath: "$.metadata.annotations[*].my-app.list[*]", indent: 10, Type: LineTypeList},
							{Path: "$.metadata.annotations[0].my-app.list[1]", GeneralizedPath: "$.metadata.annotations[*].my-app.list[*]", indent: 10, Type: LineTypeList},
							{Path: "$.metadata.annotations[0].my-app.list[2]", GeneralizedPath: "$.metadata.annotations[*].my-app.list[*]", indent: 10, Type: LineTypeList},
							{Path: "$.metadata.annotations[1].team", GeneralizedPath: "$.metadata.annotations[*].team", indent: 6, Type: LineTypeList | LineTypeMapping},
							{Path: "$.metadata.displayName", indent: 2, Type: LineTypeMapping},
						},
					},
				},
			},
		},
		"list of simple documents": {
			in: `- metadata:
- metadata:
  - foo
- metadata:
  - foo
  - bar
  - baz`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$[0].metadata", GeneralizedPath: "$.metadata", indent: 2, Type: LineTypeList | LineTypeMapping},
							{Path: "$[1].metadata", GeneralizedPath: "$.metadata", indent: 2, Type: LineTypeList | LineTypeMapping},
							{Path: "$[1].metadata[0]", GeneralizedPath: "$.metadata[*]", indent: 4, Type: LineTypeList},
							{Path: "$[2].metadata", GeneralizedPath: "$.metadata", indent: 2, Type: LineTypeList | LineTypeMapping},
							{Path: "$[2].metadata[0]", GeneralizedPath: "$.metadata[*]", indent: 4, Type: LineTypeList},
							{Path: "$[2].metadata[1]", GeneralizedPath: "$.metadata[*]", indent: 4, Type: LineTypeList},
							{Path: "$[2].metadata[2]", GeneralizedPath: "$.metadata[*]", indent: 4, Type: LineTypeList},
						},
					},
				},
			},
		},
		"list of documents": {
			in: `
- metadata:
    annotations:
      app: my-app
    displayName: my-service-2
- metadata: foo
  displayName: my-service-2
- metadata: foo
  displayName: my-service-3`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$", indent: 0, Type: LineTypeEmpty},
							{Path: "$[0].metadata", GeneralizedPath: "$.metadata", indent: 2, Type: LineTypeList | LineTypeMapping},
							{Path: "$[0].metadata.annotations", GeneralizedPath: "$.metadata.annotations", indent: 4, Type: LineTypeMapping},
							{Path: "$[0].metadata.annotations.app", GeneralizedPath: "$.metadata.annotations.app", indent: 6, Type: LineTypeMapping},
							{Path: "$[0].metadata.displayName", GeneralizedPath: "$.metadata.displayName", indent: 4, Type: LineTypeMapping},
							{Path: "$[1].metadata", GeneralizedPath: "$.metadata", indent: 2, Type: LineTypeList | LineTypeMapping},
							{Path: "$[1].displayName", GeneralizedPath: "$.displayName", indent: 2, Type: LineTypeMapping},
							{Path: "$[2].metadata", GeneralizedPath: "$.metadata", indent: 2, Type: LineTypeList | LineTypeMapping},
							{Path: "$[2].displayName", GeneralizedPath: "$.displayName", indent: 2, Type: LineTypeMapping},
						},
					},
				},
			},
		},
		"multiline string": {
			in: `
annotations:
  this
  is just a string
displayName: my-service-2`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$", indent: 0, Type: LineTypeEmpty},
							{Path: "$.annotations", indent: 0, Type: LineTypeMapping},
							{Path: "$.annotations", indent: 2, Type: LineTypeUndefined},
							{Path: "$.annotations", indent: 2, Type: LineTypeUndefined},
							{Path: "$.displayName", indent: 0, Type: LineTypeMapping},
						},
					},
				},
			},
		},
		"multiline string - block scalar | (with indented empty line)": {
			in: `
annotations: >
  long line
  
  and this is the
displayName: my-service-2`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$", indent: 0, Type: LineTypeEmpty},
							{Path: "$.annotations", indent: 0, Type: LineTypeMapping},
							{Path: "$.annotations", indent: 2, Type: LineTypeBlockScalar},
							{Path: "$.annotations", indent: 2, Type: LineTypeBlockScalar},
							{Path: "$.annotations", indent: 2, Type: LineTypeBlockScalar},
							{Path: "$.displayName", indent: 0, Type: LineTypeMapping},
						},
					},
				},
			},
		},
		"multiline string - block scalar | (with empty line)": {
			in: `
annotations: >
  long line

  and this is the
displayName: my-service-2`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$", indent: 0, Type: LineTypeEmpty},
							{Path: "$.annotations", indent: 0, Type: LineTypeMapping},
							{Path: "$.annotations", indent: 2, Type: LineTypeBlockScalar},
							{Path: "$", indent: 0, Type: LineTypeEmpty},
							// FIXME: Is this fine? Maybe!
							{Path: "$.annotations", indent: 2, Type: LineTypeUndefined},
							{Path: "$.displayName", indent: 0, Type: LineTypeMapping},
						},
					},
				},
			},
		},
		"multiline string - block scalar >": {
			in: `
annotations: >
  a
  # no comment
  b
displayName: my-service-2`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$", indent: 0, Type: LineTypeEmpty},
							{Path: "$.annotations", indent: 0, Type: LineTypeMapping},
							{Path: "$.annotations", indent: 2, Type: LineTypeBlockScalar},
							{Path: "$.annotations", indent: 2, Type: LineTypeBlockScalar},
							{Path: "$.annotations", indent: 2, Type: LineTypeBlockScalar},
							{Path: "$.displayName", indent: 0, Type: LineTypeMapping},
						},
					},
				},
			},
		},
		"multiple documents": {
			in: `---
metadata:
  name: my-service
---
metadata:
name: my-service
---`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "", indent: 0, Type: LineTypeDocSeparator},
						},
					},
					{
						Offset: 1,
						Lines: []*Line{
							{Path: "$.metadata", indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.name", indent: 2, Type: LineTypeMapping},
							{Path: "", indent: 0, Type: LineTypeDocSeparator},
						},
					},
					{
						Offset: 4,
						Lines: []*Line{
							{Path: "$.metadata", indent: 0, Type: LineTypeMapping},
							{Path: "$.name", indent: 0, Type: LineTypeMapping},
							{Path: "", indent: 0, Type: LineTypeDocSeparator},
						},
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			file := ParseFile(tc.in)
			for _, doc := range file.Docs {
				for _, line := range doc.Lines {
					line.value = ""
					line.valueColonIdx = 0
					line.listIndex = 0
				}
			}
			for _, doc := range tc.out.Docs {
				for _, line := range doc.Lines {
					if line.GeneralizedPath == "" {
						line.GeneralizedPath = line.Path
					}
				}
			}
			assert.Equal(t, tc.out, *file)
		})
	}
}

func TestDocument_FindLine(t *testing.T) {
	in := `
metadata:
  name: my-service
  project: default
spec:
  description: my-service
`
	file := ParseFile(in)

	for i, expected := range []string{
		"$",
		"$.metadata",
		"$.metadata.name",
		"$.metadata.project",
		"$.spec",
		"$.spec.description",
		"$",
	} {
		t.Run(expected, func(t *testing.T) {
			actual := file.Docs[0].FindLine(i)
			require.NotNil(t, actual)
			assert.Equal(t, expected, actual.Path)
		})
	}
}

func TestDocument_FindLineByPath(t *testing.T) {
	in := `
metadata:
  name: my-service
  project: default
spec:
  description:
  - my-service
`
	file := ParseFile(in)

	for _, tc := range []struct {
		path string
		out  string
	}{
		{"$", "$"},
		{"$.metadata", "$.metadata"},
		{"$.metadata.name", "$.metadata.name"},
		{"$.spec", "$.spec"},
		{"$.spec.description[0]", "$.spec.description[0]"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			actual := file.Docs[0].FindLineByPath(tc.path)
			require.NotNil(t, actual)
			assert.Equal(t, tc.out, actual.Path)
		})
	}
}

func TestLine_HasMapValue(t *testing.T) {
	tests := []struct {
		in       string
		line     int
		expected bool
	}{
		{"metadata:\n  name: this", 0, false},
		{"metadata:\n  name", 1, false},
		{"metadata:\n  name: this", 1, true},
		{"metadata:\n  name:", 1, false},
		{"metadata:\n  name:1", 1, false},
	}
	for _, tc := range tests {
		file := ParseFile(tc.in)
		assert.Equal(t, tc.expected, file.Docs[0].Lines[tc.line].HasMapValue())
	}
}

func TestLine_GetMapValue(t *testing.T) {
	tests := []struct {
		in       string
		line     int
		expected string
	}{
		{"metadata:\n  name: this", 0, ""},
		{"metadata:\n  name", 1, ""},
		{"metadata:\n  name: this", 1, "this"},
		{"metadata:\n  name:", 1, ""},
		{"metadata:\n  name:1", 1, ""},
	}
	for _, tc := range tests {
		file := ParseFile(tc.in)
		assert.Equal(t, tc.expected, file.Docs[0].Lines[tc.line].GetMapValue())
	}
}

func TestLine_GetMapKey(t *testing.T) {
	tests := []struct {
		in       string
		line     int
		expected string
	}{
		{"metadata:\n  name: this", 0, "metadata"},
		{"metadata:\n  name", 1, "name"},
		{"metadata:\n  name: this", 1, "name"},
		{"metadata:\n  name:", 1, "name"},
	}
	for _, tc := range tests {
		file := ParseFile(tc.in)
		assert.Equal(t, tc.expected, file.Docs[0].Lines[tc.line].GetMapKey())
	}
}

func TestLine_GetKeyPos(t *testing.T) {
	tests := []struct {
		in            string
		line          int
		expectedStart int
		expectedEnd   int
	}{
		{"metadata:\n  name: this", 0, 0, 8},
		{"metadata:\n  name", 1, 2, 6},
		{"metadata:\n  name: this", 1, 2, 6},
		{"metadata:\n  name:", 1, 2, 6},
		{"metadata:\n#  name:", 1, 0, 0},
	}
	for _, tc := range tests {
		file := ParseFile(tc.in)
		start, end := file.Docs[0].Lines[tc.line].GetKeyPos()
		assert.Equal(t, tc.expectedStart, start, "start\n%s", tc.in)
		assert.Equal(t, tc.expectedEnd, end, "end\n%s", tc.in)
	}
}

func TestLine_GetValuePos(t *testing.T) {
	tests := []struct {
		in            string
		line          int
		expectedStart int
		expectedEnd   int
	}{
		{"kind: a", 0, 6, 7},
		{"metadata", 0, 0, 0},
		{"metadata:", 0, 0, 0},
		{"metadata: this", 0, 10, 14},
		{"metadata:\n  name: this", 1, 8, 12},
		{"metadata:\n  name:  this", 1, 8, 13},
		{"metadata:\n  name:", 1, 0, 0},
	}
	for _, tc := range tests {
		file := ParseFile(tc.in)
		start, end := file.Docs[0].Lines[tc.line].GetValuePos()
		assert.Equal(t, tc.expectedStart, start, "start\n%s", tc.in)
		assert.Equal(t, tc.expectedEnd, end, "end\n%s", tc.in)
	}
}

func Test_generalizePath(t *testing.T) {
	tests := []struct {
		Input    string
		Expected string
	}{
		{Input: "", Expected: ""},
		{Input: "$", Expected: "$"},
		{Input: "$[1]", Expected: "$"},
		{Input: "$[10]", Expected: "$"},
		{Input: "$[1]", Expected: "$"},
		{Input: "$[10]", Expected: "$"},
		{Input: "$.[1]", Expected: "$.[*]"},
		{Input: "$.[10]", Expected: "$.[*]"},
		{Input: "$[5].A.B", Expected: "$.A.B"},
		{Input: "$.A[1].B[3]", Expected: "$.A[*].B[*]"},
		{Input: "$.A[10].B[300]", Expected: "$.A[*].B[*]"},
		{Input: "$[0].spec.alertMethods[0].metadata.project", Expected: "$.spec.alertMethods[*].metadata.project"},
	}
	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			assert.Equal(t, test.Expected, generalizePath(test.Input))
		})
	}
}
