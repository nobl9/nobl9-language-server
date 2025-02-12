package server

import (
	"strings"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"

	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/yamlastfast"
)

type completionProviderFunc func() []messages.CompletionItem

func newObjectsReferencesProvider(repo *objectsRepo) *objectsReferencesProvider {
	return &objectsReferencesProvider{repo: repo}
}

func (o objectsReferencesProvider) CompleteProjectName() []messages.CompletionItem {
	projects := o.repo.GetAll(manifest.KindProject)
	items := make([]messages.CompletionItem, 0, len(projects))
	for _, project := range projects {
		items = append(items, messages.CompletionItem{
			Label: project.Name,
			Kind:  messages.ReferenceCompletion,
		})
	}
	return items
}

type objectsReferencesProvider struct {
	repo *objectsRepo
}

func newPathsCompletionProvider(docs docsProvider) *pathsCompletionProvider {
	return &pathsCompletionProvider{docs: docs}
}

type pathsCompletionProvider struct {
	docs docsProvider
}

// TODO: Maybe the SDK could mark the properties that are read-only?
var excludedCompletionPaths = map[string]bool{
	"$.status":       true,
	"$.organization": true,
	"$.oktaClientID": true,
	"$.manifestSrc":  true,
}

func (p pathsCompletionProvider) Complete(
	kind manifest.Kind,
	line *yamlastfast.Line,
	position messages.Position,
) []messages.CompletionItem {
	path := normalizeRootPath(line.Path)
	if line.IsType(yamlastfast.LineTypeMapping) {
		if colonIdx := strings.Index(line.Value, ":"); colonIdx == -1 || position.Character < colonIdx {
			// Get parent path if we're at key level.
			if split := strings.Split(path, "."); len(split) > 1 {
				path = strings.Join(split[:len(split)-1], ".")
			}
		}
	}

	var proposedPaths []string
	// If we don't have a kind, we can still propose the four base paths.
	if kind == 0 {
		proposedPaths = []string{"$.apiVersion", "$.kind", "$.metadata", "$.spec"}
	} else {
		prop := p.docs.GetProperty(kind, path)
		if prop == nil {
			return nil
		}
		proposedPaths = prop.ChildrenPaths
	}
	if len(proposedPaths) == 0 {
		return nil
	}
	items := make([]messages.CompletionItem, 0, len(proposedPaths))
	for _, proposedPath := range proposedPaths {
		// Skip read-only properties.
		if excludedCompletionPaths[proposedPath] {
			continue
		}
		// Extract the last part of the path -- property name.
		if i := strings.LastIndex(proposedPath, "."); i != -1 {
			proposedPath = proposedPath[i+1:]
		}
		items = append(items, messages.CompletionItem{
			Label: proposedPath,
			Kind:  messages.PropertyCompletion,
		})
	}
	return items
}

func newCompletionProvidersRegistry(client *sdk.Client) *completionProvidersRegistry {
	repo := newObjectsRepo(client)
	refProvider := newObjectsReferencesProvider(repo)
	config := []struct {
		Path      string
		Kinds     []manifest.Kind
		Providers []completionProviderFunc
	}{
		{
			Path: "$.apiVersion",
			Providers: []completionProviderFunc{
				newStaticListProviderFunc(manifest.VersionNames()),
			},
		},
		{
			Path: "$.kind",
			Providers: []completionProviderFunc{
				newStaticListProviderFunc(manifest.KindNames()),
			},
		},
		{
			Path:  "$.metadata.project",
			Kinds: projectScopedKinds,
			Providers: []completionProviderFunc{
				refProvider.CompleteProjectName,
			},
		},
	}
	providers := make(map[string][]completionProviderFunc, len(config))
	for _, c := range config {
		if len(c.Kinds) == 0 {
			providers[c.Path] = c.Providers
			continue
		}
		for _, kind := range c.Kinds {
			providers[providerKey(kind, c.Path)] = c.Providers
		}
	}
	return &completionProvidersRegistry{providers: providers}
}

type completionProvidersRegistry struct {
	providers map[string][]completionProviderFunc
}

func (c completionProvidersRegistry) Complete(
	cmpCtx messages.CompletionContext,
	kind manifest.Kind,
	path string,
) []messages.CompletionItem {
	path = normalizeRootPath(path)
	items := make([]messages.CompletionItem, 0)
	providers := c.lookupProviders(kind, path)
	for _, providerFunc := range providers {
		items = append(items, providerFunc()...)
	}
	if charPtr := cmpCtx.TriggerCharacter; charPtr != nil && *charPtr == ":" {
		for i := range items {
			items[i].Label = " " + items[i].Label
		}
	}
	return items
}

func (c completionProvidersRegistry) lookupProviders(kind manifest.Kind, path string) []completionProviderFunc {
	providers := c.providers[providerKey(kind, path)]
	kindLessProviders := c.providers[path]
	providers = append(providers, kindLessProviders...)
	return providers
}

func providerKey(kind manifest.Kind, path string) string {
	return kind.String() + "/" + path
}

func newStaticListProviderFunc(values []string) completionProviderFunc {
	items := make([]messages.CompletionItem, 0, len(values))
	for _, value := range values {
		items = append(items, messages.CompletionItem{
			Label: value,
			Kind:  messages.ReferenceCompletion,
		})
	}
	return func() []messages.CompletionItem { return items }
}

// projectScopedKinds is a list of kinds that are scoped to a specific project.
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

// normalizeRootPath normalizes the .
func normalizeRootPath(path string) string {
	if len(path) < 2 {
		return path
	}
	if path[0] == '$' && path[1] == '[' {
		closingBracketIndex := strings.IndexRune(path, ']')
		if closingBracketIndex != -1 {
			return "$" + path[closingBracketIndex+1:]
		}
	}
	return path
}
