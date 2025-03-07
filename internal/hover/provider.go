package hover

import (
	"bytes"
	"context"
	_ "embed"
	"log/slog"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/nobl9/nobl9-go/manifest"

	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/nobl9repo"
	"github.com/nobl9/nobl9-language-server/internal/objectref"
	"github.com/nobl9/nobl9-language-server/internal/sdkdocs"
	"github.com/nobl9/nobl9-language-server/internal/yamlastsimple"
)

type docsProvider interface {
	GetProperty(kind manifest.Kind, path string) *sdkdocs.PropertyDoc
}

type objectsRepo interface {
	GetObject(ctx context.Context, kind manifest.Kind, name, project string) (manifest.Object, error)
	GetUser(ctx context.Context, id string) (*nobl9repo.User, error)
}

func NewProvider(docs docsProvider, repo objectsRepo) *Provider {
	return &Provider{
		docs: docs,
		repo: repo,
	}
}

type Provider struct {
	docs docsProvider
	repo objectsRepo
}

func (p Provider) Hover(
	ctx context.Context,
	params messages.HoverParams,
	node *files.SimpleObjectNode,
	line *yamlastsimple.Line,
) *messages.HoverResponse {
	_, keyPosEnd := line.GetKeyPos()
	var docs string
	switch {
	case params.Position.Character > keyPosEnd:
		docs = p.generatePropertyValueDoc(ctx, node, line)
		if docs == "" {
			docs = p.generatePropertyKeyDoc(ctx, node.Kind, line)
		}
	default:
		docs = p.generatePropertyKeyDoc(ctx, node.Kind, line)
	}
	if docs == "" {
		return nil
	}

	return &messages.HoverResponse{
		Contents: messages.MarkupContent{
			Kind:  messages.Markdown,
			Value: docs,
		},
	}
}

func (p Provider) generatePropertyKeyDoc(ctx context.Context, kind manifest.Kind, line *yamlastsimple.Line) string {
	prop := p.docs.GetProperty(kind, line.GeneralizedPath)
	if prop == nil {
		return ""
	}
	return p.buildPropertyDocs(ctx, prop)
}

func (p Provider) generatePropertyValueDoc(
	ctx context.Context,
	node *files.SimpleObjectNode,
	line *yamlastsimple.Line,
) string {
	if strings.TrimSpace(line.GetMapValue()) == "" {
		return ""
	}
	ref := objectref.Get(node.Kind, line)
	if ref == nil {
		return ""
	}

	switch {
	case node.Kind == manifest.KindSLO && ref.Path == "$.spec.indicator.metricSource.name":
		return p.generateSLOMetricSourceDocs(ctx, node, line, ref)
	case node.Kind == manifest.KindRoleBinding && ref.Path == "$.spec.user":
		return p.generateUserDocs(ctx, line, ref)
	case node.Kind == manifest.KindUserGroup && ref.Path == "$.spec.members[*].id":
		return p.generateUserDocs(ctx, line, ref)
	default:
		return p.generateObjectDocs(ctx, node, line, ref)
	}
}

func (p Provider) generateObjectDocs(
	ctx context.Context,
	node *files.SimpleObjectNode,
	line *yamlastsimple.Line,
	ref *objectref.Reference,
) string {
	projectName := getProjectNameFromRef(node, ref)
	if projectName == "" && objectref.IsProjectScoped(ref.Kind) {
		return ""
	}
	objectName := line.GetMapValue()
	object, err := p.repo.GetObject(ctx, ref.Kind, objectName, projectName)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get object",
			slog.String("kind", ref.Kind.String()),
			slog.String("name", objectName),
			slog.String("project", projectName),
			slog.String("error", err.Error()))
		return ""
	}
	if object == nil {
		return ""
	}
	return p.buildObjectDocs(ctx, object)
}

func (p Provider) generateSLOMetricSourceDocs(
	ctx context.Context,
	node *files.SimpleObjectNode,
	line *yamlastsimple.Line,
	ref *objectref.Reference,
) string {
	rawKind := getLineValueForPath(node, "$.spec.indicator.metricSource.kind")
	kind, err := manifest.ParseKind(rawKind)
	if err == nil {
		return ""
	}
	if kind == 0 {
		kind = manifest.KindAgent
	}
	ref.Kind = kind
	return p.generateObjectDocs(ctx, node, line, ref)
}

func (p Provider) generateUserDocs(
	ctx context.Context,
	line *yamlastsimple.Line,
	ref *objectref.Reference,
) string {
	userID := line.GetMapValue()
	user, err := p.repo.GetUser(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get user",
			slog.String("kind", ref.Kind.String()),
			slog.String("userID", userID),
			slog.String("error", err.Error()))
		return ""
	}
	if user == nil {
		return ""
	}
	tpl := userDocTpl()
	var b bytes.Buffer
	if err = tpl.Execute(&b, user); err != nil {
		slog.ErrorContext(ctx, "failed to execute user doc template",
			slog.String("kind", ref.Kind.String()),
			slog.String("userID", userID),
			slog.String("error", err.Error()))
		return ""
	}
	return b.String()
}

func (p Provider) buildObjectDocs(ctx context.Context, object manifest.Object) string {
	objectYAML, err := yaml.Marshal(object)
	if err != nil {
		slog.ErrorContext(ctx, "failed to encode object to YAML format",
			slog.String("kind", object.GetKind().String()),
			slog.String("name", object.GetName()),
			slog.String("error", err.Error()))
		return ""
	}
	objectYAMLStr := string(objectYAML)

	tpl := objectDocTpl()
	var b bytes.Buffer
	if err = tpl.Execute(&b, objectDocData{
		Kind:        object.GetKind(),
		Name:        object.GetName(),
		Description: findObjectDescription(ctx, objectYAMLStr),
		YAML:        objectYAMLStr,
	}); err != nil {
		slog.ErrorContext(ctx, "failed to execute object doc template",
			slog.String("kind", object.GetKind().String()),
			slog.String("name", object.GetName()),
			slog.String("error", err.Error()))
		return ""
	}
	return b.String()
}

func findObjectDescription(ctx context.Context, rawObject string) string {
	file, err := files.ParseSimpleObjectFile(rawObject)
	if err != nil {
		slog.ErrorContext(ctx, "failed to parse object YAML",
			slog.String("error", err.Error()))
		return ""
	}
	if len(file) == 0 {
		slog.ErrorContext(ctx, "unexpected YAML parsing result")
		return ""
	}
	descriptionLine := file[0].FindLineByPath("$.spec.description")
	if descriptionLine == nil {
		return ""
	}
	v := descriptionLine.GetMapValue()
	if v == `""` {
		return ""
	}
	return v
}

func (p Provider) buildPropertyDocs(ctx context.Context, doc *sdkdocs.PropertyDoc) string {
	lastDot := strings.LastIndex(doc.Path, ".")
	propertyName := doc.Path[lastDot+1:]
	rules := filterSlice(doc.Rules, func(rule sdkdocs.RulePlan) bool {
		return rule.Description != "" && rule.Description != "TODO"
	})

	tpl := propertyDocTpl()
	var b bytes.Buffer
	if err := tpl.Execute(&b, propertyDocData{
		Name:     propertyName,
		Type:     doc.Type,
		Doc:      doc.Doc,
		Rules:    rules,
		Examples: doc.Examples,
	}); err != nil {
		slog.ErrorContext(ctx, "failed to execute property doc template",
			slog.String("path", doc.Path),
			slog.String("error", err.Error()))
		return ""
	}
	return b.String()
}

func getProjectNameFromRef(node *files.SimpleObjectNode, ref *objectref.Reference) string {
	if ref.ProjectPath == "" {
		return ""
	}
	projectName := getLineValueForPath(node, ref.ProjectPath)
	if projectName == "" {
		fallbackProjectPath := ref.FallbackProjectPath(ref.Kind)
		projectName = getLineValueForPath(node, fallbackProjectPath)
	}
	return projectName
}

func getLineValueForPath(node *files.SimpleObjectNode, path string) string {
	line := node.FindLineByPath(path)
	if line == nil {
		return ""
	}
	return line.GetMapValue()
}

func filterSlice[T any](s []T, f func(T) bool) []T {
	if len(s) == 0 {
		return nil
	}
	var res []T
	for _, v := range s {
		if f(v) {
			res = append(res, v)
		}
	}
	return res
}
