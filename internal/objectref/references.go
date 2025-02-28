package objectref

import (
	"github.com/nobl9/nobl9-go/manifest"

	"github.com/nobl9/nobl9-language-server/internal/yamlpath"
)

func Find(kind manifest.Kind, path string) *Reference {
	refs, ok := objectReferences[kind]
	if !ok {
		return nil
	}
	for _, ref := range refs {
		if yamlpath.Match(ref.Path, path) {
			return ref
		}
	}
	return nil
}

func Get(kind manifest.Kind) []*Reference {
	refs, _ := objectReferences[kind]
	return refs
}

type Reference struct {
	Path        string
	ProjectPath string
	Kind        manifest.Kind
	AppliesTo   []manifest.Kind
}

var objectReferences = func() map[manifest.Kind][]*Reference {
	references := []Reference{
		{
			Path:      "$.metadata.project",
			Kind:      manifest.KindProject,
			AppliesTo: projectScopedKinds,
		},
		// Only auto completion.
		//{
		//	Path:      "$.metadata.labels",
		//	AppliesTo: labelsSupportingKinds,
		//},
		{
			Path:        "$.spec.alertMethods[*].name",
			ProjectPath: "$.spec.alertMethods[*].project",
			Kind:        manifest.KindAlertMethod,
			AppliesTo:   []manifest.Kind{manifest.KindAlertPolicy},
		},
		{
			Path:      "$.spec.alertMethods[*].project",
			Kind:      manifest.KindAlertMethod,
			AppliesTo: []manifest.Kind{manifest.KindAlertPolicy},
		},
		{
			Path:        "$.spec.slo",
			ProjectPath: "$.metadata.project",
			Kind:        manifest.KindSLO,
			AppliesTo:   []manifest.Kind{manifest.KindAlertSilence},
		},
		{
			Path:        "$.spec.alertPolicy.name",
			ProjectPath: "$.spec.alertPolicy.project",
			Kind:        manifest.KindAlertPolicy,
			AppliesTo:   []manifest.Kind{manifest.KindAlertSilence},
		},
		{
			Path:      "$.spec.alertPolicy.project",
			Kind:      manifest.KindProject,
			AppliesTo: []manifest.Kind{manifest.KindAlertSilence},
		},
		{
			Path:        "$.spec.slo",
			ProjectPath: "$.metadata.project",
			Kind:        manifest.KindSLO,
			AppliesTo:   []manifest.Kind{manifest.KindAnnotation},
		},
		{
			Path:      "$.spec.objectiveName",
			AppliesTo: []manifest.Kind{manifest.KindAnnotation},
		},
		{
			Path:        "$.spec.filters[*].slos[*].name",
			ProjectPath: "$.spec.filters[*].slos[*].project",
			Kind:        manifest.KindSLO,
			AppliesTo:   []manifest.Kind{manifest.KindBudgetAdjustment},
		},
		{
			Path:      "$.spec.filters[*].slos[*].project",
			Kind:      manifest.KindSLO,
			AppliesTo: []manifest.Kind{manifest.KindBudgetAdjustment},
		},
		{
			Path:      "$.spec.filters.projects[*]",
			Kind:      manifest.KindProject,
			AppliesTo: []manifest.Kind{manifest.KindReport},
		},
		{
			Path:        "$.spec.filters.services[*].name",
			ProjectPath: "$.spec.filters.services[*].project",
			Kind:        manifest.KindService,
			AppliesTo:   []manifest.Kind{manifest.KindReport},
		},
		{
			Path:      "$.spec.filters.services[*].project",
			Kind:      manifest.KindProject,
			AppliesTo: []manifest.Kind{manifest.KindReport},
		},
		{
			Path:        "$.spec.filters.slos[*].name",
			ProjectPath: "$.spec.filters.slos[*].project",
			Kind:        manifest.KindSLO,
			AppliesTo:   []manifest.Kind{manifest.KindReport},
		},
		{
			Path:      "$.spec.filters.slos[*].project",
			Kind:      manifest.KindProject,
			AppliesTo: []manifest.Kind{manifest.KindReport},
		},
		//{
		//	Path:      "$.spec.filters.labels",
		//	AppliesTo: []manifest.Kind{manifest.KindReport},
		//},
		{
			Path:      "$.spec.user",
			AppliesTo: []manifest.Kind{manifest.KindRoleBinding},
		},
		{
			Path:      "$.spec.groupRef",
			Kind:      manifest.KindUserGroup,
			AppliesTo: []manifest.Kind{manifest.KindRoleBinding},
		},
		{
			Path:      "$.spec.roleRef",
			AppliesTo: []manifest.Kind{manifest.KindRoleBinding},
		},
		{
			Path:      "$.spec.projectRef",
			Kind:      manifest.KindProject,
			AppliesTo: []manifest.Kind{manifest.KindRoleBinding},
		},
		{
			Path:      "$.spec.members[*].id",
			AppliesTo: []manifest.Kind{manifest.KindUserGroup},
		},
		{
			Path:        "$.spec.service",
			ProjectPath: "$.metadata.project",
			Kind:        manifest.KindService,
			AppliesTo:   []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:      "$.spec.indicator.metricSource.name",
			AppliesTo: []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:      "$.spec.indicator.metricSource.project",
			Kind:      manifest.KindProject,
			AppliesTo: []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:        "$.spec.alertPolicies[*]",
			ProjectPath: "$.metadata.project",
			Kind:        manifest.KindAlertPolicy,
			AppliesTo:   []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:      "$.spec.objectives[*].composite.components.objectives[*].project",
			Kind:      manifest.KindProject,
			AppliesTo: []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:        "$.spec.objectives[*].composite.components.objectives[*].slo",
			ProjectPath: "$.spec.objectives[*].composite.components.objectives[*].project",
			Kind:        manifest.KindSLO,
			AppliesTo:   []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:      "$.spec.objectives[*].composite.components.objectives[*].objective",
			AppliesTo: []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:        "$.spec.anomalyConfig.noData.alertMethods[*].name",
			ProjectPath: "$.spec.anomalyConfig.noData.alertMethods[*].project",
			Kind:        manifest.KindAlertMethod,
			AppliesTo:   []manifest.Kind{manifest.KindSLO},
		},
		{
			Path:      "$.spec.anomalyConfig.noData.alertMethods[*].project",
			Kind:      manifest.KindProject,
			AppliesTo: []manifest.Kind{manifest.KindSLO},
		},
	}
	m := make(map[manifest.Kind][]*Reference)
	for _, r := range references {
		for _, kind := range r.AppliesTo {
			m[kind] = append(m[kind], &r)
		}
	}
	return m
}()

// TODO: generate that from docs.
var projectScopedKinds = []manifest.Kind{
	manifest.KindSLO,
	manifest.KindService,
	manifest.KindAgent,
	manifest.KindAlertPolicy,
	manifest.KindAlertSilence,
	manifest.KindProject,
	manifest.KindAlertMethod,
	manifest.KindDirect,
	manifest.KindDataExport,
	manifest.KindAnnotation,
}

// TODO: generate that from docs.
var labelsSupportingKinds = []manifest.Kind{
	manifest.KindAlertPolicy,
	manifest.KindProject,
	manifest.KindService,
	manifest.KindSLO,
}
