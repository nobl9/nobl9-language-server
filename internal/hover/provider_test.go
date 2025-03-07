package hover

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nobl9/nobl9-language-server/internal/sdkdocs"
)

func TestProvider_buildPropertyDocs(t *testing.T) {
	provider := Provider{}

	tests := map[string]struct {
		in  sdkdocs.PropertyDoc
		out string
	}{
		"minimal": {
			in: sdkdocs.PropertyDoc{
				Path: "foo",
				Type: "string",
			},
			out: "`foo:string`",
		},
		"infer name from path": {
			in: sdkdocs.PropertyDoc{
				Path: "$.foo[*].bar",
				Type: "string",
			},
			out: "`bar:string`",
		},
		"with doc": {
			in: sdkdocs.PropertyDoc{
				Path: "$.bar",
				Type: "string",
				Doc:  "bar is a string",
			},
			out: "`bar:string`\n\nbar is a string",
		},
		"with a single rule": {
			in: sdkdocs.PropertyDoc{
				Path: "$.bar",
				Type: "string",
				Rules: []sdkdocs.RulePlan{
					{Description: "must be a string"},
				},
			},
			out: "`bar:string`\n\n**Validation rules:**\n\n- must be a string",
		},
		"with two rules": {
			in: sdkdocs.PropertyDoc{
				Path: "$.bar",
				Type: "string",
				Rules: []sdkdocs.RulePlan{
					{Description: "must be a string"},
					{Description: "must be above the bar"},
				},
			},
			out: "`bar:string`\n\n**Validation rules:**\n\n- must be a string\n- must be above the bar",
		},
		"rule with details": {
			in: sdkdocs.PropertyDoc{
				Path: "$.bar",
				Type: "string",
				Rules: []sdkdocs.RulePlan{
					{Description: "must be a string", Details: "some details"},
				},
			},
			out: "`bar:string`\n\n**Validation rules:**\n\n- must be a string; some details",
		},
		"rule with conditions": {
			in: sdkdocs.PropertyDoc{
				Path: "$.bar",
				Type: "string",
				Rules: []sdkdocs.RulePlan{
					{Description: "must be a string", Conditions: []string{"some condition", "another condition"}},
				},
			},
			out: "`bar:string`\n\n**Validation rules:**\n\n- must be a string\n" +
				"  Conditions:\n  - some condition\n  - another condition",
		},
		"with a single example": {
			in: sdkdocs.PropertyDoc{
				Path:     "$.bar",
				Type:     "string",
				Examples: []string{"example 1"},
			},
			out: "`bar:string`\n\n**Examples:**\n\n```yaml\nexample 1\n```",
		},
		"with two examples": {
			in: sdkdocs.PropertyDoc{
				Path:     "$.bar",
				Type:     "string",
				Examples: []string{"example: 1", "example: 2"},
			},
			out: "`bar:string`\n\n**Examples:**\n\n```yaml\nexample: 1\n---\nexample: 2\n```",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := provider.buildPropertyDocs(context.Background(), &test.in)
			assert.Equal(t, test.out, result)
		})
	}
}
