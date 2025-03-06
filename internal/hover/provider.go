package hover

import (
	"context"
	_ "embed"
	"fmt"
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
	GetUsers(ctx context.Context, phrase string) ([]*nobl9repo.User, error)
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
			docs = p.generatePropertyKeyDoc(node.Kind, line)
		}
	default:
		docs = p.generatePropertyKeyDoc(node.Kind, line)
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

func (p Provider) generatePropertyKeyDoc(kind manifest.Kind, line *yamlastsimple.Line) string {
	prop := p.docs.GetProperty(kind, line.GeneralizedPath)
	if prop == nil {
		return ""
	}
	return p.buildDocs(prop)
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
	//case ref.SLOPath != "":
	//return p.completeObjectiveNames(ctx, node, ref)
	//case node.Kind == manifest.KindSLO && ref.Path == "$.spec.indicator.metricSource.name":
	//return p.completeSLOMetricSourceName(ctx, node, ref)
	//case node.Kind == manifest.KindRoleBinding && ref.Path == "$.spec.user":
	//return p.completeUserIDs(ctx, line)
	//case node.Kind == manifest.KindRoleBinding && ref.Path == "$.spec.roleRef":
	//return p.completeRoleBindingRoles(ctx, node)
	//case node.Kind == manifest.KindUserGroup && ref.Path == "$.spec.members[*].id":
	//return p.completeUserIDs(ctx, line)
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

	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("`%s` %s", object.GetName(), object.GetKind()))
	if description := findObjectDescription(ctx, objectYAMLStr); description != "" {
		b.WriteString("\n\n")
		b.WriteString(description)
	}
	b.WriteString("\n\n")
	b.WriteString("```yaml\n")
	b.WriteString(objectYAMLStr)
	b.WriteString("```")
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
	return descriptionLine.GetMapValue()
}

func (p Provider) buildDocs(doc *sdkdocs.PropertyDoc) string {
	b := strings.Builder{}
	lastDot := strings.LastIndex(doc.Path, ".")
	propertyName := doc.Path[lastDot+1:]
	b.WriteString(fmt.Sprintf("`%s:%s`", propertyName, doc.Type))
	if doc.Doc != "" {
		b.WriteString("\n\n")
		b.WriteString(doc.Doc)
	}
	rules := filterSlice(doc.Rules, func(rule sdkdocs.RulePlan) bool {
		return rule.Description != "" && rule.Description != "TODO"
	})
	if len(rules) > 0 {
		b.WriteString("\n\n**Validation rules:**")
		for _, rule := range rules {
			b.WriteString("\n")
			b.WriteString("- ")
			b.WriteString(markdownEscape(rule.Description))
			if rule.Details != "" {
				b.WriteString("; ")
				b.WriteString(markdownEscape(rule.Details))
			}
			if len(rule.Conditions) > 0 {
				b.WriteString("\n  Conditions:\n")
				for _, condition := range rule.Conditions {
					b.WriteString("    - ")
					b.WriteString(markdownEscape(condition))
					b.WriteString("\n")
				}
			}
		}
	}
	if len(doc.Examples) > 0 {
		b.WriteString("\n\n**Examples:**\n")
		b.WriteString("\n```yaml\n")
		for i, example := range doc.Examples {
			b.WriteString(example)
			if i < len(doc.Examples)-1 {
				b.WriteString("\n---\n")
			}
		}
		b.WriteString("\n```")
	}
	return b.String()
}

// Based on: https://github.com/mattcone/markdown-guide/blob/master/_basic-syntax/escaping-characters.md
const markdownSpecialCharacters = "\\`*_{}[]<>()#+-.!|"

var markdownReplacer = func() *strings.Replacer {
	var replacements []string
	for _, c := range markdownSpecialCharacters {
		replacements = append(replacements, string(c), `\`+string(c))
	}
	return strings.NewReplacer(replacements...)
}()

// markdownEscape escapes markdown characters in the given string.
func markdownEscape(s string) string {
	return markdownReplacer.Replace(s)
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
