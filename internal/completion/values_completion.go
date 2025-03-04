package completion

import (
	"context"

	"github.com/nobl9/nobl9-go/manifest"

	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/yamlastsimple"
)

func NewValuesCompletionProvider(docs docsProvider) *ValuesCompletionProvider {
	return &ValuesCompletionProvider{docs: docs}
}

type ValuesCompletionProvider struct {
	docs docsProvider
}

func (p ValuesCompletionProvider) getType() completionProviderType {
	return valuesCompletionType
}

func (p ValuesCompletionProvider) Complete(
	_ context.Context,
	_ messages.CompletionParams,
	_ files.SimpleObjectFile,
	node *files.SimpleObjectNode,
	line *yamlastsimple.Line,
) []messages.CompletionItem {
	path := line.GeneralizedPath

	var values []string
	switch path {
	case "$.kind":
		values = manifest.KindNames()
	case "$.apiVersion":
		values = manifest.VersionNames()
	default:
		propDoc := p.docs.GetProperty(node.Kind, path)
		if propDoc == nil || len(propDoc.Values) == 0 {
			return nil
		}
		values = propDoc.Values
	}

	items := make([]messages.CompletionItem, 0, len(values))
	for _, value := range values {
		items = append(items, messages.CompletionItem{
			Label: value,
			Kind:  messages.ValueCompletion,
		})
	}
	return items
}
