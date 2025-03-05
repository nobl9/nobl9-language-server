package completion

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nobl9/nobl9-go/manifest"
	v1alphaSLO "github.com/nobl9/nobl9-go/manifest/v1alpha/slo"

	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/nobl9repo"
	"github.com/nobl9/nobl9-language-server/internal/objectref"
	"github.com/nobl9/nobl9-language-server/internal/sdkdocs"
	"github.com/nobl9/nobl9-language-server/internal/yamlastsimple"
)

type objectsRepo interface {
	GetAllNames(ctx context.Context, kind manifest.Kind, project string) []string
	GetObject(ctx context.Context, kind manifest.Kind, name, project string) (manifest.Object, error)
	GetUsers(ctx context.Context, phrase string) ([]*nobl9repo.User, error)
	GetRoles(ctx context.Context) (*nobl9repo.Roles, error)
}

type docsProvider interface {
	GetProperty(kind manifest.Kind, path string) *sdkdocs.PropertyDoc
}

func NewReferencesCompletionProvider(repo objectsRepo) *ReferencesCompletionProvider {
	return &ReferencesCompletionProvider{repo: repo}
}

type ReferencesCompletionProvider struct {
	repo objectsRepo
}

func (p ReferencesCompletionProvider) getType() completionProviderType {
	return valuesCompletionType
}

func (p ReferencesCompletionProvider) Complete(
	ctx context.Context,
	params messages.CompletionParams,
	_ files.SimpleObjectFile,
	node *files.SimpleObjectNode,
	line *yamlastsimple.Line,
) []messages.CompletionItem {
	ref := objectref.Get(node.Kind, line)
	if ref == nil {
		return nil
	}

	var items []messages.CompletionItem
	switch {
	case ref.SLOPath != "":
		items = p.completeObjectiveNames(ctx, node, ref)
	case node.Kind == manifest.KindSLO && ref.Path == "$.spec.indicator.metricSource.name":
		items = p.completeSLOMetricSourceName(ctx, node, ref)
	case node.Kind == manifest.KindRoleBinding && ref.Path == "$.spec.user":
		items = p.completeUserIDs(ctx, line)
	case node.Kind == manifest.KindRoleBinding && ref.Path == "$.spec.roleRef":
		items = p.completeRoleBindingRoles(ctx, node)
	case node.Kind == manifest.KindUserGroup && ref.Path == "$.spec.members[*].id":
		items = p.completeUserIDs(ctx, line)
	default:
		items = p.completeObjectNames(ctx, node, ref)
	}

	if charPtr := params.CompletionContext.TriggerCharacter; charPtr != nil && *charPtr == ":" {
		for i := range items {
			items[i].Label = " " + items[i].Label
		}
	}
	return items
}

func (p ReferencesCompletionProvider) completeObjectNames(
	ctx context.Context,
	node *files.SimpleObjectNode,
	ref *objectref.Reference,
) []messages.CompletionItem {
	projectName := getProjectNameFromRef(node, ref)
	if projectName == "" && objectref.IsProjectScoped(ref.Kind) {
		return nil
	}
	names := p.repo.GetAllNames(ctx, ref.Kind, projectName)
	items := make([]messages.CompletionItem, 0, len(names))
	for i := range names {
		items = append(items, messages.CompletionItem{
			Label: names[i],
			Kind:  messages.ReferenceCompletion,
		})
	}
	return items
}

func (p ReferencesCompletionProvider) completeUserIDs(
	ctx context.Context,
	line *yamlastsimple.Line,
) []messages.CompletionItem {
	users, err := p.repo.GetUsers(ctx, line.GetMapValue())
	if err != nil {
		slog.ErrorContext(ctx, "failed to get users", slog.String("error", err.Error()))
		return nil
	}
	items := make([]messages.CompletionItem, 0, len(users))
	for _, user := range users {
		items = append(items, messages.CompletionItem{
			Label:            fmt.Sprintf("%s %s (%s)", user.FirstName, user.LastName, user.Email),
			Kind:             messages.ReferenceCompletion,
			InsertText:       user.UserID,
			InsertTextFormat: messages.PlainTextTextFormat,
		})
	}
	return items
}

func (p ReferencesCompletionProvider) completeRoleBindingRoles(
	ctx context.Context,
	node *files.SimpleObjectNode,
) []messages.CompletionItem {
	rolesResp, err := p.repo.GetRoles(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get roles", slog.String("error", err.Error()))
		return nil
	}
	var roles []nobl9repo.Role
	if node.FindLineByPath("$.spec.projectRef") == nil {
		roles = append(roles, rolesResp.OrganizationRoles...)
	} else {
		roles = append(roles, rolesResp.ProjectRoles...)
	}
	items := make([]messages.CompletionItem, 0, len(roles))
	for i := range roles {
		items = append(items, messages.CompletionItem{
			Label: roles[i].Name,
			Kind:  messages.ReferenceCompletion,
		})
	}
	return items
}

func (p ReferencesCompletionProvider) completeSLOMetricSourceName(
	ctx context.Context,
	node *files.SimpleObjectNode,
	ref *objectref.Reference,
) []messages.CompletionItem {
	rawKind := getLineValueForPath(node, "$.spec.indicator.metricSource.kind")
	kind, err := manifest.ParseKind(rawKind)
	if err == nil {
		return nil
	}
	if kind == 0 {
		kind = manifest.KindAgent
	}
	ref.Kind = kind
	return p.completeObjectNames(ctx, node, ref)
}

func (p ReferencesCompletionProvider) completeObjectiveNames(
	ctx context.Context,
	node *files.SimpleObjectNode,
	ref *objectref.Reference,
) []messages.CompletionItem {
	sloName := getLineValueForPath(node, ref.SLOPath)
	if sloName == "" {
		return nil
	}
	projectName := getProjectNameFromRef(node, ref)
	if projectName == "" {
		return nil
	}
	object, err := p.repo.GetObject(ctx, manifest.KindSLO, sloName, projectName)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get SLO object", slog.String("error", err.Error()))
		return nil
	}
	slo, ok := object.(v1alphaSLO.SLO)
	if !ok {
		slog.ErrorContext(ctx, fmt.Sprintf("failed to cast %T object to %T", object, slo))
		return nil
	}
	items := make([]messages.CompletionItem, 0, len(slo.Spec.Objectives))
	for i := range slo.Spec.Objectives {
		items = append(items, messages.CompletionItem{
			Label: slo.Spec.Objectives[i].Name,
			Kind:  messages.ReferenceCompletion,
		})
	}
	return items
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
