package diagnostics

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha"
	v1alphaAlertPolicy "github.com/nobl9/nobl9-go/manifest/v1alpha/alertpolicy"
	v1alphaAlertSilence "github.com/nobl9/nobl9-go/manifest/v1alpha/alertsilence"
	v1alphaAnnotation "github.com/nobl9/nobl9-go/manifest/v1alpha/annotation"
	v1alphaBudgetAdjustment "github.com/nobl9/nobl9-go/manifest/v1alpha/budgetadjustment"
	v1alphaReport "github.com/nobl9/nobl9-go/manifest/v1alpha/report"
	v1alphaRoleBinding "github.com/nobl9/nobl9-go/manifest/v1alpha/rolebinding"
	v1alphaSLO "github.com/nobl9/nobl9-go/manifest/v1alpha/slo"
	"github.com/pkg/errors"

	"github.com/nobl9/nobl9-language-server/internal/config"
	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/yamlast"
	"github.com/nobl9/nobl9-language-server/internal/yamlastsimple"
	"github.com/nobl9/nobl9-language-server/internal/yamlpath"
)

const goYamlSource = "go-yaml"

type deprecatedPathsProvider interface {
	GetDeprecatedPaths(kind manifest.Kind) []string
}

type objectsProvider interface {
	GetObject(ctx context.Context, kind manifest.Kind, name, project string) (manifest.Object, error)
	GetDefaultProject() string
}

func NewProvider(deprecated deprecatedPathsProvider, objects objectsProvider) *Provider {
	return &Provider{
		deprecated: deprecated,
		objects:    objects,
	}
}

type Provider struct {
	deprecated deprecatedPathsProvider
	objects    objectsProvider
}

func (d Provider) DiagnoseFile(ctx context.Context, file *files.File) []messages.Diagnostic {
	if file.Err != nil {
		return astErrorToDiagnostics(file.Err, 0)
	}
	var diagnostics []messages.Diagnostic
	for _, object := range file.Objects {
		if object.Err != nil {
			diagnostics = append(diagnostics, astErrorToDiagnostics(object.Err, object.Node.StartLine)...)
			continue
		}
		diagnostics = append(diagnostics, d.validateObject(ctx, object)...)
	}
	for _, object := range file.SimpleAST {
		diagnostics = append(diagnostics, d.checkDeprecated(object)...)
	}
	return diagnostics
}

func (d Provider) validateObject(ctx context.Context, object *files.ObjectNode) []messages.Diagnostic {
	var diagnostics []messages.Diagnostic
	err := object.Object.Validate()
	if err == nil {
		diagnostics = append(diagnostics, d.checkReferencedObjects(ctx, object)...)
		return diagnostics
	}
	var oErr *v1alpha.ObjectError
	if ok := errors.As(err, &oErr); ok {
		if oDiags := objectValidationErrorToDiagnostics(ctx, oErr, object.Node); len(oDiags) > 0 {
			diagnostics = append(diagnostics, oDiags...)
		}
	} else {
		diagnostics = append(diagnostics, messages.Diagnostic{
			Range:    newLineRange(object.Node.StartLine, 0, 0),
			Severity: messages.DiagnosticSeverityError,
			Source:   ptr(config.ServerName),
			Message:  err.Error(),
		})
	}
	return diagnostics
}

func (d Provider) checkReferencedObjects(ctx context.Context, object *files.ObjectNode) []messages.Diagnostic {
	if object.Object == nil {
		return nil
	}
	diagnostics := d.checkForProjectExistence(ctx, object)
	switch v := object.Object.(type) {
	case v1alphaSLO.SLO:
		diagnostics = append(diagnostics, d.checkSLOReferencedObjects(ctx, object, v)...)
	case v1alphaAlertPolicy.AlertPolicy:
		diagnostics = append(diagnostics, d.checkAlertPolicyReferencedObjects(ctx, object, v)...)
	case v1alphaAlertSilence.AlertSilence:
		diagnostics = append(diagnostics, d.checkAlertSilenceReferencedObjects(ctx, object, v)...)
	case v1alphaAnnotation.Annotation:
		diagnostics = append(diagnostics, d.checkAnnotationReferencedObjects(ctx, object, v)...)
	case v1alphaBudgetAdjustment.BudgetAdjustment:
		diagnostics = append(diagnostics, d.checkBudgetAdjustmentReferencedObjects(ctx, object, v)...)
	case v1alphaReport.Report:
		diagnostics = append(diagnostics, d.checkReportReferencedObjects(ctx, object, v)...)
	case v1alphaRoleBinding.RoleBinding:
		diagnostics = append(diagnostics, d.checkRoleBindingReferencedObjects(ctx, object, v)...)
	}
	return diagnostics
}

func (d Provider) checkRoleBindingReferencedObjects(
	ctx context.Context,
	object *files.ObjectNode,
	roleBinding v1alphaRoleBinding.RoleBinding,
) []messages.Diagnostic {
	return nil
}

func (d Provider) checkReportReferencedObjects(
	ctx context.Context,
	object *files.ObjectNode,
	report v1alphaReport.Report,
) []messages.Diagnostic {
	return nil
}

func (d Provider) checkBudgetAdjustmentReferencedObjects(
	ctx context.Context,
	object *files.ObjectNode,
	budgetAdjustment v1alphaBudgetAdjustment.BudgetAdjustment,
) []messages.Diagnostic {
	var diagnostics []messages.Diagnostic
	for i, sloRef := range budgetAdjustment.Spec.Filters.SLOs {
		diags := d.checkObjectExistence(
			ctx,
			object.Node,
			fmt.Sprintf("$.spec.filters.slos[%d].project", i),
			manifest.KindProject,
			sloRef.Project,
			"",
		)
		if len(diags) > 0 {
			diagnostics = append(diagnostics, diags...)
			continue
		}
		diagnostics = append(diagnostics, d.checkObjectExistence(
			ctx,
			object.Node,
			fmt.Sprintf("$.spec.filters.slos[%d].name", i),
			manifest.KindSLO,
			sloRef.Name,
			sloRef.Project,
		)...)
	}
	return diagnostics
}

func (d Provider) checkAnnotationReferencedObjects(
	ctx context.Context,
	object *files.ObjectNode,
	annotation v1alphaAnnotation.Annotation,
) []messages.Diagnostic {
	diags := d.checkObjectExistence(
		ctx,
		object.Node,
		"$.spec.slo",
		manifest.KindSLO,
		annotation.Spec.Slo,
		annotation.GetProject(),
	)
	if len(diags) > 0 {
		return diags
	}
	return d.checkObjectiveExistence(
		ctx,
		object.Node,
		"$.spec.objective",
		annotation.Spec.ObjectiveName,
		annotation.Spec.Slo,
		annotation.GetProject(),
	)
}

func (d Provider) checkAlertSilenceReferencedObjects(
	ctx context.Context,
	object *files.ObjectNode,
	alertSilence v1alphaAlertSilence.AlertSilence,
) []messages.Diagnostic {
	alertPolicyProject := alertSilence.Spec.AlertPolicy.Project
	if alertPolicyProject == "" {
		alertPolicyProject = alertSilence.GetProject()
	} else {
		diags := d.checkObjectExistence(
			ctx,
			object.Node,
			"$.spec.alertPolicy.project",
			manifest.KindProject,
			alertPolicyProject,
			"",
		)
		if len(diags) > 0 {
			return diags
		}
	}
	diags := d.checkObjectExistence(
		ctx,
		object.Node,
		"$.spec.alertPolicy.name",
		manifest.KindAlertPolicy,
		alertSilence.Spec.AlertPolicy.Name,
		alertPolicyProject,
	)
	diags = append(diags, d.checkObjectExistence(
		ctx,
		object.Node,
		"$.spec.slo",
		manifest.KindSLO,
		alertSilence.Spec.SLO,
		alertSilence.GetProject(),
	)...)
	return diags
}

func (d Provider) checkAlertPolicyReferencedObjects(
	ctx context.Context,
	object *files.ObjectNode,
	alertPolicy v1alphaAlertPolicy.AlertPolicy,
) []messages.Diagnostic {
	var diagnostics []messages.Diagnostic
	for i, alertMethod := range alertPolicy.Spec.AlertMethods {
		project := alertMethod.Metadata.Project
		if project == "" {
			project = alertPolicy.GetProject()
		} else {
			diags := d.checkObjectExistence(
				ctx,
				object.Node,
				fmt.Sprintf("$.spec.alertMethods[%d].metadata.project", i),
				manifest.KindProject,
				project,
				"",
			)
			if len(diags) > 0 {
				diagnostics = append(diagnostics, diags...)
				continue
			}
		}
		diagnostics = append(diagnostics, d.checkObjectExistence(
			ctx,
			object.Node,
			fmt.Sprintf("$.spec.alertMethods[%d].metadata.name", i),
			manifest.KindAlertMethod,
			alertMethod.Metadata.Name,
			project,
		)...)
	}
	return diagnostics
}

func (d Provider) checkSLOReferencedObjects(
	ctx context.Context,
	object *files.ObjectNode,
	slo v1alphaSLO.SLO,
) []messages.Diagnostic {
	var diagnostics []messages.Diagnostic
	if slo.Spec.Service != "" {
		diagnostics = append(diagnostics, d.checkObjectExistence(
			ctx,
			object.Node,
			"$.spec.service",
			manifest.KindService,
			slo.Spec.Service,
			slo.GetProject(),
		)...)
	}
	for i, alertPolicy := range slo.Spec.AlertPolicies {
		if alertPolicy == "" {
			continue
		}
		diagnostics = append(diagnostics, d.checkObjectExistence(
			ctx,
			object.Node,
			fmt.Sprintf("$.spec.alertPolicies[%d]", i),
			manifest.KindAlertPolicy,
			alertPolicy,
			slo.GetProject(),
		)...)
	}
	if slo.Spec.Indicator != nil {
		diagnostics = append(diagnostics, d.checkSLOIndicatorReferencedObjects(ctx, object, slo)...)
	}
	if slo.Spec.AnomalyConfig != nil && slo.Spec.AnomalyConfig.NoData != nil {
		diagnostics = append(diagnostics, d.checkSLOAnomalyConfigReferencedObjects(ctx, object, slo)...)
	}
	diagnostics = append(diagnostics, d.checkSLOCompositeReferencedObjects(ctx, object, slo)...)
	return diagnostics
}

func (d Provider) checkSLOIndicatorReferencedObjects(
	ctx context.Context,
	object *files.ObjectNode,
	slo v1alphaSLO.SLO,
) []messages.Diagnostic {
	sourceProject := slo.Spec.Indicator.MetricSource.Project
	if sourceProject == "" {
		sourceProject = slo.GetProject()
	} else {
		diags := d.checkObjectExistence(
			ctx,
			object.Node,
			"$.spec.indicator.metricSource.project",
			manifest.KindProject,
			sourceProject,
			"",
		)
		if len(diags) > 0 {
			return diags
		}
	}
	sourceKind := slo.Spec.Indicator.MetricSource.Kind
	if sourceKind == 0 {
		sourceKind = manifest.KindAgent
	}
	return d.checkObjectExistence(
		ctx,
		object.Node,
		"$.spec.indicator.metricSource.name",
		sourceKind,
		slo.Spec.Indicator.MetricSource.Name,
		sourceProject,
	)
}

func (d Provider) checkSLOAnomalyConfigReferencedObjects(
	ctx context.Context,
	object *files.ObjectNode,
	slo v1alphaSLO.SLO,
) []messages.Diagnostic {
	var diagnostics []messages.Diagnostic
	for i, alertMethod := range slo.Spec.AnomalyConfig.NoData.AlertMethods {
		project := alertMethod.Project
		if project == "" {
			project = slo.GetProject()
		} else {
			diags := d.checkObjectExistence(
				ctx,
				object.Node,
				fmt.Sprintf("$.spec.anomalyConfig.noData.alertMethods[%d].project", i),
				manifest.KindProject,
				project,
				"",
			)
			if len(diags) > 0 {
				diagnostics = append(diagnostics, diags...)
				continue
			}
		}
		diagnostics = append(diagnostics, d.checkObjectExistence(
			ctx,
			object.Node,
			fmt.Sprintf("$.spec.anomalyConfig.noData.alertMethods[%d].name", i),
			manifest.KindAlertMethod,
			alertMethod.Name,
			project,
		)...)
	}
	return diagnostics
}

func (d Provider) checkSLOCompositeReferencedObjects(
	ctx context.Context,
	object *files.ObjectNode,
	slo v1alphaSLO.SLO,
) []messages.Diagnostic {
	var diagnostics []messages.Diagnostic
	for i := range slo.Spec.Objectives {
		if slo.Spec.Objectives[i].Composite == nil {
			continue
		}
		for j, objective := range slo.Spec.Objectives[i].Composite.Objectives {
			diags := d.checkObjectExistence(
				ctx,
				object.Node,
				fmt.Sprintf("$.spec.objectives[%d].composite.components.objectives[%d].project", i, j),
				manifest.KindProject,
				objective.Project,
				"",
			)
			if len(diags) > 0 {
				diagnostics = append(diagnostics, diags...)
				continue
			}
			diags = d.checkObjectExistence(
				ctx,
				object.Node,
				fmt.Sprintf("$.spec.objectives[%d].composite.components.objectives[%d].slo", i, j),
				manifest.KindSLO,
				objective.SLO,
				objective.Project,
			)
			if len(diags) > 0 {
				diagnostics = append(diagnostics, diags...)
				continue
			}
			diags = d.checkObjectiveExistence(
				ctx,
				object.Node,
				fmt.Sprintf("$.spec.objectives[%d].composite.components.objectives[%d].objective", i, j),
				objective.Objective,
				objective.SLO,
				objective.Project,
			)
			if len(diags) > 0 {
				diagnostics = append(diagnostics, diags...)
				continue
			}
		}
	}
	return diagnostics
}

func (d Provider) checkForProjectExistence(ctx context.Context, object *files.ObjectNode) []messages.Diagnostic {
	projectScopedObject, ok := object.Object.(manifest.ProjectScopedObject)
	if !ok {
		return nil
	}
	project := projectScopedObject.GetProject()
	if project == "" {
		return nil
	}
	return d.checkObjectExistence(
		ctx,
		object.Node,
		"$.metadata.project",
		manifest.KindProject,
		project,
		"",
	)
}

func (d Provider) checkObjectExistence(
	ctx context.Context,
	node *yamlast.Node,
	propertyPath string,
	kind manifest.Kind,
	objectName, projectName string,
) []messages.Diagnostic {
	object, err := d.objects.GetObject(ctx, kind, objectName, projectName)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"failed to fetch object for reference check",
			slog.Any("error", err),
			slog.String("kind", kind.String()),
			slog.String("propPath", propertyPath),
			slog.Any("objectName", objectName),
			slog.Any("projectName", projectName),
		)
		return nil
	}
	if object != nil {
		return nil
	}
	mappingValueNode := findNodeForPath(ctx, propertyPath, node.Node)
	if mappingValueNode == nil {
		return nil
	}
	var message string
	if projectName != "" {
		message = fmt.Sprintf("%s does not exist in Project %s", kind, projectName)
	} else {
		message = fmt.Sprintf("%s does not exist", kind)
	}
	return []messages.Diagnostic{{
		Range:    getRangeFromNode(mappingValueNode),
		Severity: messages.DiagnosticSeverityError,
		Source:   ptr(config.ServerName),
		Message:  message,
	}}
}

func (d Provider) checkObjectiveExistence(
	ctx context.Context,
	node *yamlast.Node,
	propertyPath string,
	objectiveName, sloName, projectName string,
) []messages.Diagnostic {
	object, err := d.objects.GetObject(ctx, manifest.KindSLO, sloName, projectName)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"failed to fetch SLO for reference check",
			slog.Any("error", err),
			slog.String("propPath", propertyPath),
			slog.Any("sloName", sloName),
			slog.Any("projectName", projectName),
		)
		return nil
	}
	if object == nil {
		return nil
	}
	slo, ok := object.(v1alphaSLO.SLO)
	if !ok {
		slog.ErrorContext(ctx, "failed to cast object to SLO")
		return nil
	}
	for i := range slo.Spec.Objectives {
		if slo.Spec.Objectives[i].Name == objectiveName {
			return nil
		}
	}
	mappingValueNode := findNodeForPath(ctx, propertyPath, node.Node)
	if mappingValueNode == nil {
		return nil
	}
	return []messages.Diagnostic{{
		Range:    getRangeFromNode(mappingValueNode),
		Severity: messages.DiagnosticSeverityError,
		Source:   ptr(config.ServerName),
		Message: fmt.Sprintf(
			"objective does not exist in SLO %s and Project %s",
			sloName, projectName),
	}}
}

func findNodeForPath(ctx context.Context, path string, node ast.Node) ast.Node {
	p, err := yamlpath.FromString(path)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get yaml path",
			slog.String("path", path),
			slog.Any("error", err))
		return nil
	}
	filteredNode, _, err := p.FilterNode(node)
	if err != nil {
		if !errors.Is(err, yaml.ErrNotFoundNode) {
			slog.ErrorContext(ctx, "failed to read yaml node by path",
				slog.Any("yamlPath", p),
				slog.String("propPath", path),
				slog.Any("error", err))
		}
		return nil
	}
	if filteredNode == nil {
		slog.ErrorContext(ctx, "failed to find yaml node by path - node is nil")
		return nil
	}
	return filteredNode
}

func (d Provider) checkDeprecated(object *files.SimpleObjectNode) []messages.Diagnostic {
	// TODO: Cache the paths!!! There's no need to recompute them every time since they are static.
	paths := d.deprecated.GetDeprecatedPaths(object.Kind)
	var diagnostics []messages.Diagnostic
	for _, path := range paths {
		for i, line := range object.Doc.Lines {
			if !line.IsType(yamlastsimple.LineTypeMapping) {
				continue
			}
			if !yamlpath.Match(line.Path, path) {
				continue
			}
			start, end := line.GetKeyPos()
			diagnostics = append(diagnostics, messages.Diagnostic{
				Range:    newLineRange(object.Doc.Offset+i+1, start, end),
				Severity: messages.DiagnosticSeverityWarning,
				Source:   ptr(config.ServerName),
				Message:  "property is deprecated",
			})
		}
	}
	return diagnostics
}

func objectValidationErrorToDiagnostics(
	ctx context.Context,
	err *v1alpha.ObjectError,
	node *yamlast.Node,
) []messages.Diagnostic {
	var diagnostics []messages.Diagnostic
	for _, propErr := range err.Errors {
		// TODO: Consider caching the path.
		yamlPath := "$"
		if propErr.PropertyName != "" {
			propName := propErr.PropertyName
			yamlPath += "." + propName
		}
		p, err := yamlpath.FromString(yamlPath)
		if err != nil {
			slog.ErrorContext(ctx, "failed to get yaml path", slog.Any("error", err))
			continue
		}
		var rng messages.Range
		filteredNode, firstMatch, err := p.FilterNode(node.Node)
		if err != nil {
			if !errors.Is(err, yaml.ErrNotFoundNode) {
				slog.ErrorContext(ctx, "failed to read yaml node by path",
					slog.Any("yamlPath", p),
					slog.String("propName", propErr.PropertyName),
					slog.Any("error", err))
				continue
			}
			rng = newPointRange(node.StartLine, 0)
		} else {
			rng = getRangeFromNode(filteredNode)
		}
		for _, ruleErr := range propErr.Errors {
			var msg string
			// We only want to add property name to the error message
			// if we manage to pinpoint the diagnostic to the specific value.
			if !firstMatch && propErr.PropertyName != "" {
				msg = propErr.PropertyName + ": " + ruleErr.Message
			} else {
				msg = ruleErr.Message
			}
			diagnostics = append(diagnostics, messages.Diagnostic{
				Range:    rng,
				Severity: messages.DiagnosticSeverityError,
				Source:   ptr(config.ServerName),
				Message:  msg,
			})
		}
	}
	return diagnostics
}

func astErrorToDiagnostics(err error, line int) []messages.Diagnostic {
	diagnostics := make([]messages.Diagnostic, 0)
	if tErr := yaml.AsTokenScopedError(err); tErr != nil {
		diagnostics = append(diagnostics, messages.Diagnostic{
			Range: newPointRange(
				// Shift the line number to the actual line in the file as the SDK
				// operates on the single node's context.
				// Subtract one to convert StartLine from 1-based to 0-based indexing.
				tErr.Token.Position.Line,
				tErr.Token.Position.Column,
			),
			Severity: messages.DiagnosticSeverityError,
			Source:   ptr(goYamlSource),
			Message:  tErr.Msg,
		})
	} else {
		diagnostics = append(diagnostics, messages.Diagnostic{
			Range:    newPointRange(line, 0),
			Severity: messages.DiagnosticSeverityError,
			Source:   ptr(config.ServerName),
			Message:  err.Error(),
		})
	}
	return diagnostics
}

func getRangeFromNode(node ast.Node) messages.Range {
	switch v := node.(type) {
	case *ast.MappingValueNode:
		if v.Value.GetPath() != node.GetPath() || v.Value.Type() == ast.NullType {
			// If the value is another mapping, or it's empty we want to highlight the key.
			// Example:
			//
			//   period:
			//     startTime: 2025-01-01T12:00:00+02:00
			token := v.Key.GetToken()
			return newLineRange(token.Position.Line, token.Position.Column-1, v.Start.Position.Column-1)
		} else {
			// If the value is a simple type we highlight the value.
			// Example:
			//
			//   name: foo
			token := v.Value.GetToken()
			return newLineRange(token.Position.Line, token.Position.Column-1, len(node.String()))
		}
	case *ast.StringNode:
		token := node.GetToken()
		return newLineRange(token.Position.Line, token.Position.Column-1, token.Position.Column+len(v.Value)-1)
	default:
		token := node.GetToken()
		return newLineRange(token.Position.Line, token.Position.IndentNum, token.Position.Column)
	}
}

// newPointRange returns [messages.Range] with both start and end
// set to the same line and character.
func newPointRange(l, c int) messages.Range {
	return newRange(l, c, l, c)
}

// newPointRange returns [messages.Range] with both start and end
// set to the same line.
func newLineRange(l, sc, ec int) messages.Range {
	return newRange(l, sc, l, ec)
}

// Line numbers are 0-based in the LSP specification, but we operate on 1-based indexing,
// so we need to subtract one.
func newRange(sl, sc, el, ec int) messages.Range {
	if sl != 0 {
		sl--
	}
	if el != 0 {
		el--
	}
	return messages.Range{
		Start: messages.Position{
			Line:      sl,
			Character: sc,
		},
		End: messages.Position{
			Line:      el,
			Character: ec,
		},
	}
}

func ptr[T any](v T) *T { return &v }
