package completion

import (
	"context"
	"strings"

	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/yamlastsimple"
)

func NewKeysCompletionProvider(docs docsProvider) *KeysCompletionProvider {
	return &KeysCompletionProvider{docs: docs}
}

type KeysCompletionProvider struct {
	docs docsProvider
}

// TODO: Maybe the SDK could mark the properties that are read-only?
var excludedCompletionPaths = map[string]bool{
	"$.status":       true,
	"$.organization": true,
	"$.oktaClientID": true,
	"$.manifestSrc":  true,
}

func (p KeysCompletionProvider) getType() completionProviderType {
	return keysCompletionType
}

func (p KeysCompletionProvider) Complete(
	_ context.Context,
	_ messages.CompletionParams,
	_ files.SimpleObjectFile,
	node *files.SimpleObjectNode,
	line *yamlastsimple.Line,
) []messages.CompletionItem {
	path := line.GeneralizedPath
	// Get parent path if the key has been completed.
	if line.IsType(yamlastsimple.LineTypeMapping) {
		if split := strings.Split(path, "."); len(split) > 1 {
			path = strings.Join(split[:len(split)-1], ".")
		}
	}

	var proposedPaths []string
	// If we don't have a kind, we can still propose the four base paths.
	if node.Kind == 0 {
		proposedPaths = []string{"$.apiVersion", "$.kind", "$.metadata", "$.spec"}
	} else {
		prop := p.docs.GetProperty(node.Kind, path)
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
		prop := p.docs.GetProperty(node.Kind, path+"."+proposedPath)
		var insertText string
		isRootSpecOrMetadata := path == "$" && (proposedPath == "spec" || proposedPath == "metadata")
		switch {
		case (prop == nil || len(prop.ChildrenPaths) == 0) && !isRootSpecOrMetadata:
			// Proposed path is a simple value node.
			insertText = proposedPath + ": "
		case strings.HasSuffix(proposedPath, "[*]"):
			insertText = strings.TrimSuffix(proposedPath, "[*]") + ":\n" + strings.Repeat(" ", line.GetIndent()) + "- "
		default:
			// Proposed path has children nodes and therefore continues on the next line.
			// FIXME: This needs to be handled better, we should either detect default
			// tabstop character or use workspace/configuration or expose LS config option.
			insertText = proposedPath + ":\n" + strings.Repeat(" ", line.GetIndent()+2)
		}
		items = append(items, messages.CompletionItem{
			Label:            proposedPath,
			Kind:             messages.PropertyCompletion,
			InsertText:       insertText,
			InsertTextFormat: messages.PlainTextTextFormat,
		})
	}
	return items
}
