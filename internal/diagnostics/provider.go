package diagnostics

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha"
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
		if v.Spec.Service != "" {
			diag, exists := d.checkObjectExistence(
				ctx,
				object.Node,
				"$.spec.service",
				manifest.KindService,
				v.Spec.Service,
				v.GetProject(),
			)
			if !exists {
				diagnostics = append(diagnostics, diag)
			}
		}
		for i, alertPolicy := range v.Spec.AlertPolicies {
			if alertPolicy == "" {
				continue
			}
			diag, exists := d.checkObjectExistence(
				ctx,
				object.Node,
				"$.spec.alertPolicies["+strconv.Itoa(i)+"]",
				manifest.KindAlertPolicy,
				alertPolicy,
				v.GetProject(),
			)
			if !exists {
				diagnostics = append(diagnostics, diag)
			}
		}
		if v.Spec.Indicator != nil {
			sourceProject := v.Spec.Indicator.MetricSource.Project
			if sourceProject == "" {
				sourceProject = v.GetProject()
			}
			sourceKind := v.Spec.Indicator.MetricSource.Kind
			if sourceKind == 0 {
				sourceKind = manifest.KindAgent
			}
			diag, exists := d.checkObjectExistence(
				ctx,
				object.Node,
				"$.spec.indicator.metricSource",
				sourceKind,
				v.Spec.Indicator.MetricSource.Name,
				sourceProject,
			)
			if !exists {
				diagnostics = append(diagnostics, diag)
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
	diag, exists := d.checkObjectExistence(
		ctx,
		object.Node,
		"$.metadata.project",
		manifest.KindProject,
		project,
		"",
	)
	if !exists {
		return []messages.Diagnostic{diag}
	}
	return nil
}

func (d Provider) checkObjectExistence(
	ctx context.Context,
	node *yamlast.Node,
	propertyPath string,
	kind manifest.Kind,
	objectName, projectName string,
) (diag messages.Diagnostic, exists bool) {
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
		return diag, true
	}
	if object != nil {
		return diag, true
	}
	p, err := yamlpath.FromString(propertyPath)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get yaml path", slog.Any("error", err))
		return diag, true
	}
	var rng messages.Range
	filteredNode, _, err := p.FilterNode(node.Node)
	if err != nil {
		if !errors.Is(err, yaml.ErrNotFoundNode) {
			slog.ErrorContext(ctx, "failed to read yaml node by path",
				slog.Any("yamlPath", p),
				slog.String("propPath", propertyPath),
				slog.Any("error", err))
			return diag, true
		}
		rng = newPointRange(node.StartLine, 0)
	} else {
		rng = getRangeFromNode(filteredNode)
	}
	var message string
	if projectName != "" {
		message = fmt.Sprintf("%s does not exist in Project %s", kind, projectName)
	} else {
		message = fmt.Sprintf("%s does not exist", kind)
	}
	return messages.Diagnostic{
		Range:    rng,
		Severity: messages.DiagnosticSeverityError,
		Source:   ptr(config.ServerName),
		Message:  message,
	}, false
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
		return newLineRange(token.Position.Line, 0, token.Position.Column)
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
