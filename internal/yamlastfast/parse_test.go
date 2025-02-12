package yamlastfast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
							{Path: "$.metadata", Indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.name", Indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.annotations", Indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.annotations.app", Indent: 4, Type: LineTypeMapping},
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
							{Path: "$.metadata", Indent: 0, Type: LineTypeMapping},
							{Path: "$.app", Indent: 0, Type: LineTypeMapping},
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
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
							{Path: "$.metadata", Indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.name", Indent: 2, Type: LineTypeMapping},
							{Path: "", Indent: 0, Type: LineTypeComment},
							{Path: "$.metadata.name.app", Indent: 4, Type: LineTypeMapping},
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
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
							{Path: "$.metadata", Indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.name", Indent: 2, Type: LineTypeMapping},
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
							{Path: "$.metadata", Indent: 2, Type: LineTypeEmpty},
							{Path: "$.metadata.name.app", Indent: 4, Type: LineTypeMapping},
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
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
							{Path: "$.metadata", Indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.name", Indent: 2, Type: LineTypeMapping},
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
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
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
							{Path: "$.apiVersion", Indent: 0, Type: LineTypeMapping},
							{Path: "$.kind", Indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata", Indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.name", Indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.annotations", Indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.annotations.app", Indent: 4, Type: LineTypeMapping},
							{Path: "$.metadata.displayName", Indent: 2, Type: LineTypeMapping},
							{Path: "$.spec", Indent: 0, Type: LineTypeMapping},
							{Path: "$.spec.description", Indent: 2, Type: LineTypeMapping},
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
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
							{Path: "$.apiVersion", Indent: 0, Type: LineTypeMapping},
							{Path: "$.kind", Indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata", Indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.name", Indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.annotations", Indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.annotations.app", Indent: 4, Type: LineTypeMapping},
							{Path: "$.metadata.labels", Indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.labels.team[*]", Indent: 4, Type: LineTypeMapping},
							{Path: "$.metadata.labels.team[0]", Indent: 8, Type: LineTypeList},
							{Path: "$.metadata.labels.team[1]", Indent: 8, Type: LineTypeList},
							{Path: "$.metadata.displayName", Indent: 2, Type: LineTypeMapping},
							{Path: "$.spec", Indent: 0, Type: LineTypeMapping},
							{Path: "$.spec.description", Indent: 2, Type: LineTypeMapping},
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
							{Path: "$.labels", Indent: 0, Type: LineTypeMapping},
							{Path: "$.labels.team[*]", Indent: 2, Type: LineTypeMapping},
							{Path: "$.labels.team[0]", Indent: 4, Type: LineTypeList},
							{Path: "$.labels.team[1]", Indent: 4, Type: LineTypeList},
							{Path: "$.displayName", Indent: 0, Type: LineTypeMapping},
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
    - team: green
  displayName: my-service-1`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
							{Path: "$.metadata", Indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.annotations[*]", Indent: 2, Type: LineTypeMapping},
							{Path: "$.metadata.annotations[0].app", Indent: 6, Type: LineTypeList | LineTypeMapping},
							{Path: "$.metadata.annotations[0].my-app", Indent: 6, Type: LineTypeMapping},
							{Path: "$.metadata.annotations[0].my-app.foo", Indent: 8, Type: LineTypeMapping},
							{Path: "$.metadata.annotations[0].my-app.list[*]", Indent: 8, Type: LineTypeMapping},
							{Path: "$.metadata.annotations[0].my-app.list[0]", Indent: 10, Type: LineTypeList},
							{Path: "$.metadata.annotations[0].my-app.list[1]", Indent: 10, Type: LineTypeList},
							{Path: "$.metadata.annotations[1].team", Indent: 6, Type: LineTypeList | LineTypeMapping},
							{Path: "$.metadata.displayName", Indent: 2, Type: LineTypeMapping},
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
  displayName: my-service-1`,
			out: File{
				Docs: []*Document{
					{
						Lines: []*Line{
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
							{Path: "$[0].metadata", Indent: 2, Type: LineTypeList | LineTypeMapping},
							{Path: "$[0].metadata.annotations", Indent: 4, Type: LineTypeMapping},
							{Path: "$[0].metadata.annotations.app", Indent: 6, Type: LineTypeMapping},
							{Path: "$[0].metadata.displayName", Indent: 4, Type: LineTypeMapping},
							{Path: "$[1].metadata", Indent: 2, Type: LineTypeList | LineTypeMapping},
							{Path: "$[1].displayName", Indent: 2, Type: LineTypeMapping},
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
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
							{Path: "$.annotations", Indent: 0, Type: LineTypeMapping},
							{Path: "$.annotations", Indent: 2, Type: LineTypeUndefined},
							{Path: "$.annotations", Indent: 2, Type: LineTypeUndefined},
							{Path: "$.displayName", Indent: 0, Type: LineTypeMapping},
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
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
							{Path: "$.annotations", Indent: 0, Type: LineTypeMapping},
							{Path: "$.annotations", Indent: 2, Type: LineTypeBlockScalar},
							{Path: "$.annotations", Indent: 2, Type: LineTypeBlockScalar},
							{Path: "$.annotations", Indent: 2, Type: LineTypeBlockScalar},
							{Path: "$.displayName", Indent: 0, Type: LineTypeMapping},
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
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
							{Path: "$.annotations", Indent: 0, Type: LineTypeMapping},
							{Path: "$.annotations", Indent: 2, Type: LineTypeBlockScalar},
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
							// FIXME: Is this fine? Maybe!
							{Path: "$.annotations", Indent: 2, Type: LineTypeUndefined},
							{Path: "$.displayName", Indent: 0, Type: LineTypeMapping},
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
							{Path: "$", Indent: 0, Type: LineTypeEmpty},
							{Path: "$.annotations", Indent: 0, Type: LineTypeMapping},
							{Path: "$.annotations", Indent: 2, Type: LineTypeBlockScalar},
							{Path: "$.annotations", Indent: 2, Type: LineTypeBlockScalar},
							{Path: "$.annotations", Indent: 2, Type: LineTypeBlockScalar},
							{Path: "$.displayName", Indent: 0, Type: LineTypeMapping},
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
							{Path: "", Indent: 0, Type: LineTypeDocSeparator},
						},
					},
					{
						Offset: 1,
						Lines: []*Line{
							{Path: "$.metadata", Indent: 0, Type: LineTypeMapping},
							{Path: "$.metadata.name", Indent: 2, Type: LineTypeMapping},
							{Path: "", Indent: 0, Type: LineTypeDocSeparator},
						},
					},
					{
						Offset: 4,
						Lines: []*Line{
							{Path: "$.metadata", Indent: 0, Type: LineTypeMapping},
							{Path: "$.name", Indent: 0, Type: LineTypeMapping},
							{Path: "", Indent: 0, Type: LineTypeDocSeparator},
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
					line.Value = "" // Clear value for comparison.
				}
			}
			assert.Equal(t, tc.out, *file)
		})
	}
}

func TestFile_FindLine(t *testing.T) {
	in := `---
metadata:
  name: my-service
---
metadata:
name: my-service
---`
	file := ParseFile(in)

	for i, line := range []string{
		"",
		"$.metadata",
		"$.metadata.name",
		"",
		"$.metadata",
		"$.name",
		"",
	} {
		assert.Equal(t, line, file.FindLine(i).Path)
	}
}

func TestLine_GetMapValue(t *testing.T) {
	in := `metadata:
  name: my-service`
	file := ParseFile(in)

	assert.Equal(t, "my-service", file.Docs[0].Lines[1].GetMapValue())
}

func TestLine_GetMapKey(t *testing.T) {
	in := `metadata:
  name: my-service`
	file := ParseFile(in)

	assert.Equal(t, "name", file.Docs[0].Lines[1].GetMapKey())
}
