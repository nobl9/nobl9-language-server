package completion

import (
	"context"

	"github.com/nobl9/nobl9-go/manifest"

	"github.com/nobl9/nobl9-language-server/internal/messages"
)

type ObjectsRefProvider struct {
	repo objectsRepo
}

func NewObjectsRefProvider(repo objectsRepo) *ObjectsRefProvider {
	return &ObjectsRefProvider{repo: repo}
}

func (o ObjectsRefProvider) CompleteProjectName(ctx context.Context) []messages.CompletionItem {
	projects := o.repo.GetAllNames(ctx, manifest.KindProject, "")
	items := make([]messages.CompletionItem, 0, len(projects))
	for _, project := range projects {
		items = append(items, messages.CompletionItem{
			Label: project,
			Kind:  messages.ReferenceCompletion,
		})
	}
	return items
}
