package objectref

import (
	"testing"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/stretchr/testify/assert"

	"github.com/nobl9/nobl9-language-server/internal/yamlastsimple"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name     string
		kind     manifest.Kind
		line     *yamlastsimple.Line
		expected *Reference
	}{
		{
			kind: manifest.KindAlertPolicy,
			line: &yamlastsimple.Line{
				Path:            "$.spec.alertMethods[2].metadata.project",
				GeneralizedPath: "$.spec.alertMethods[*].metadata.project",
			},
			expected: &Reference{
				Kind: manifest.KindProject,
				Path: "$.spec.alertMethods[*].metadata.project",
			},
		},
		{
			kind: manifest.KindAlertPolicy,
			line: &yamlastsimple.Line{
				Path:            "$.spec.alertMethods[2].metadata.name",
				GeneralizedPath: "$.spec.alertMethods[*].metadata.name",
			},
			expected: &Reference{
				Kind:        manifest.KindAlertMethod,
				Path:        "$.spec.alertMethods[*].metadata.name",
				ProjectPath: "$.spec.alertMethods[2].metadata.project",
			},
		},
		{
			kind: manifest.KindReport,
			line: &yamlastsimple.Line{
				Path:            "$.spec.filters.slos[0].name",
				GeneralizedPath: "$.spec.filters.slos[*].name",
			},
			expected: &Reference{
				Kind:        manifest.KindSLO,
				Path:        "$.spec.filters.slos[*].name",
				ProjectPath: "$.spec.filters.slos[0].project",
			},
		},
		{
			kind: manifest.KindSLO,
			line: &yamlastsimple.Line{
				Path:            "$.spec.objectives[2].composite.components.objectives[1].objective",
				GeneralizedPath: "$.spec.objectives[*].composite.components.objectives[*].objective",
			},
			expected: &Reference{
				Kind:        manifest.KindSLO,
				Path:        "$.spec.objectives[*].composite.components.objectives[*].objective",
				SLOPath:     "$.spec.objectives[2].composite.components.objectives[1].slo",
				ProjectPath: "$.spec.objectives[2].composite.components.objectives[1].project",
			},
		},
		{
			kind: manifest.KindService,
			line: &yamlastsimple.Line{
				Path:            "$.non.existent.path",
				GeneralizedPath: "$.non.existent.path",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.line.Path, func(t *testing.T) {
			result := Get(tt.kind, tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReference_FallbackProjectPath(t *testing.T) {
	tests := []struct {
		name     string
		kind     manifest.Kind
		ref      Reference
		expected string
	}{
		{
			name: "AlertPolicy with non-project path",
			kind: manifest.KindAlertPolicy,
			ref: Reference{
				Path: "$.spec.alertMethods[2].metadata.name",
			},
			expected: "$.metadata.project",
		},
		{
			name: "AlertPolicy with project path",
			kind: manifest.KindAlertPolicy,
			ref: Reference{
				Kind: manifest.KindAlertPolicy,
				Path: "$.metadata.project",
			},
			expected: "",
		},
		{
			name: "Report with project path",
			kind: manifest.KindReport,
			ref: Reference{
				Path: "$.spec.filters.services[0].name",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ref.FallbackProjectPath(tt.kind)
			assert.Equal(t, tt.expected, result)
		})
	}
}
