package objectref

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/nobl9/nobl9-go/manifest"

	"github.com/nobl9/nobl9-language-server/internal/yamlastsimple"
)

// Get returns a reference for the given kind and generalized line path.
func Get(kind manifest.Kind, line *yamlastsimple.Line) *Reference {
	refs, ok := objectReferences[kind]
	if !ok {
		return nil
	}
	ref, ok := refs[line.GeneralizedPath]
	if !ok {
		return nil
	}
	result := &Reference{
		Kind: ref.Kind,
		Path: ref.Path,
	}
	if ref.ProjectPath == "" && ref.SLOPath == "" {
		return result
	}
	linePath := line.Path
	if strings.HasPrefix(linePath, "$[") {
		closingBracketIdx := strings.Index(linePath, "]")
		if closingBracketIdx != -1 {
			linePath = linePath[closingBracketIdx+1:]
		}
	}
	if ref.ProjectPath != "" {
		result.ProjectPath = calculateReferencedPath(linePath, ref.ProjectPath)
	}
	if ref.SLOPath != "" {
		result.SLOPath = calculateReferencedPath(linePath, ref.SLOPath)
	}
	return result
}

type Reference struct {
	Kind manifest.Kind
	// TODO: better names!
	Path        string
	ProjectPath string
	SLOPath     string

	appliesTo []manifest.Kind
}

func (r Reference) FallbackProjectPath(kind manifest.Kind) string {
	if r.Path != "$.metadata.project" && slices.Contains(projectScopedKinds, kind) {
		return "$.metadata.project"
	}
	return ""
}

var objectReferences = func() map[manifest.Kind]map[string]*Reference {
	references := []Reference{
		{
			Path:      "$.metadata.project",
			Kind:      manifest.KindProject,
			appliesTo: projectScopedKinds,
		},
		{
			Path:        "$.spec.alertMethods[*].metadata.name",
			ProjectPath: "$.spec.alertMethods[%d].metadata.project",
			Kind:        manifest.KindAlertMethod,
			appliesTo:   []manifest.Kind{manifest.KindAlertPolicy},
		},
		{
			Path:      "$.spec.alertMethods[*].metadata.project",
			Kind:      manifest.KindProject,
			appliesTo: []manifest.Kind{manifest.KindAlertPolicy},
		},
		{
			Path:        "$.spec.slo",
			ProjectPath: "$.metadata.project",
			Kind:        manifest.KindSLO,
			appliesTo:   []manifest.Kind{manifest.KindAlertSilence},
		},
		{
			Path:        "$.spec.alertPolicy.name",
			ProjectPath: "$.spec.alertPolicy.project",
			Kind:        manifest.KindAlertPolicy,
			appliesTo:   []manifest.Kind{manifest.KindAlertSilence},
		},
		{
			Path:      "$.spec.alertPolicy.project",
			Kind:      manifest.KindProject,
			appliesTo: []manifest.Kind{manifest.KindAlertSilence},
		},
		{
			Path:        "$.spec.slo",
			ProjectPath: "$.metadata.project",
			Kind:        manifest.KindSLO,
			appliesTo:   []manifest.Kind{manifest.KindAnnotation},
		},
		{
			Path:        "$.spec.objectiveName",
			SLOPath:     "$.spec.slo",
			ProjectPath: "$.metadata.project",
			appliesTo:   []manifest.Kind{manifest.KindAnnotation},
		},
		{
			Path:        "$.spec.filters.slos[*].name",
			ProjectPath: "$.spec.filters.slos[%d].project",
			Kind:        manifest.KindSLO,
			appliesTo:   []manifest.Kind{manifest.KindBudgetAdjustment},
		},
		{
			Path:      "$.spec.filters.slos[*].project",
			Kind:      manifest.KindSLO,
			appliesTo: []manifest.Kind{manifest.KindBudgetAdjustment},
		},
		{
			Path:      "$.spec.filters.projects[*]",
			Kind:      manifest.KindProject,
			appliesTo: []manifest.Kind{manifest.KindReport},
		},
		{
			Path:        "$.spec.filters.services[*].name",
			ProjectPath: "$.spec.filters.services[%d].project",
			Kind:        manifest.KindService,
			appliesTo:   []manifest.Kind{manifest.KindReport},
		},
		{
			Path:      "$.spec.filters.services[*].project",
			Kind:      manifest.KindProject,
			appliesTo: []manifest.Kind{manifest.KindReport},
		},
		{
			Path:        "$.spec.filters.slos[*].name",
			ProjectPath: "$.spec.filters.slos[%d].project",
			Kind:        manifest.KindSLO,
			appliesTo:   []manifest.Kind{manifest.KindReport},
		},
		{
			Path:      "$.spec.filters.slos[*].project",
			Kind:      manifest.KindProject,
			appliesTo: []manifest.Kind{manifest.KindReport},
		},
		{
			Path:      "$.spec.user",
			appliesTo: []manifest.Kind{manifest.KindRoleBinding},
		},
		{
			Path:      "$.spec.groupRef",
			Kind:      manifest.KindUserGroup,
			appliesTo: []manifest.Kind{manifest.KindRoleBinding},
		},
		{
			Path:      "$.spec.roleRef",
			appliesTo: []manifest.Kind{manifest.KindRoleBinding},
		},
		{
			Path:      "$.spec.projectRef",
			Kind:      manifest.KindProject,
			appliesTo: []manifest.Kind{manifest.KindRoleBinding},
		},
		{
			Path:      "$.spec.members[*].id",
			appliesTo: []manifest.Kind{manifest.KindUserGroup},
		},
		{
			Path:        "$.spec.service",
			ProjectPath: "$.metadata.project",
			Kind:        manifest.KindService,
			appliesTo:   []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:        "$.spec.indicator.metricSource.name",
			ProjectPath: "$.spec.indicator.metricSource.project",
			appliesTo:   []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:      "$.spec.indicator.metricSource.project",
			Kind:      manifest.KindProject,
			appliesTo: []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:        "$.spec.alertPolicies",
			ProjectPath: "$.metadata.project",
			Kind:        manifest.KindAlertPolicy,
			appliesTo:   []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:      "$.spec.objectives[*].composite.components.objectives[*].project",
			Kind:      manifest.KindProject,
			appliesTo: []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:        "$.spec.objectives[*].composite.components.objectives[*].slo",
			ProjectPath: "$.spec.objectives[%d].composite.components.objectives[%d].project",
			Kind:        manifest.KindSLO,
			appliesTo:   []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:        "$.spec.objectives[*].composite.components.objectives[*].objective",
			SLOPath:     "$.spec.objectives[%d].composite.components.objectives[%d].slo",
			ProjectPath: "$.spec.objectives[%d].composite.components.objectives[%d].project",
			Kind:        manifest.KindSLO,
			appliesTo:   []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:        "$.spec.anomalyConfig.noData.alertMethods[*].name",
			ProjectPath: "$.spec.anomalyConfig.noData.alertMethods[%d].project",
			Kind:        manifest.KindAlertMethod,
			appliesTo:   []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:      "$.spec.anomalyConfig.noData.alertMethods[*].project",
			Kind:      manifest.KindProject,
			appliesTo: []manifest.Kind{manifest.KindSLO},
		},
	}
	m := make(map[manifest.Kind]map[string]*Reference)
	for _, r := range references {
		for _, kind := range r.appliesTo {
			if _, ok := m[kind]; !ok {
				m[kind] = make(map[string]*Reference)
			}
			m[kind][r.Path] = &r
		}
	}
	return m
}()

func IsProjectScoped(kind manifest.Kind) bool {
	return slices.Contains(projectScopedKinds, kind)
}

// TODO: generate that from docs.
var projectScopedKinds = []manifest.Kind{
	manifest.KindSLO,
	manifest.KindService,
	manifest.KindAgent,
	manifest.KindAlertPolicy,
	manifest.KindAlertSilence,
	manifest.KindAlertMethod,
	manifest.KindDirect,
	manifest.KindDataExport,
	manifest.KindAnnotation,
}

func calculateReferencedPath(basePath, refPath string) string {
	var (
		arrStart bool
		nStr     string
	)
	indexes := make([]any, 0)
	for _, ch := range basePath {
		switch {
		case ch == '[':
			arrStart = true
		case ch == ']':
			arrStart = false
			n, _ := strconv.Atoi(nStr)
			indexes = append(indexes, n)
			nStr = ""
		case arrStart:
			nStr += string(ch)
		}
	}
	if len(indexes) == 0 {
		return refPath
	}
	return fmt.Sprintf(refPath, indexes...)
}
