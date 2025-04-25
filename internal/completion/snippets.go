package completion

import (
	"context"
	_ "embed"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/yamlastsimple"
)

//go:embed snippets.json
var snippetsJSONData []byte

func NewSnippetsProvider() *SnippetsProvider {
	return &SnippetsProvider{items: setupSnippetItems()}
}

type SnippetsProvider struct {
	items []messages.CompletionItem
}

func (p SnippetsProvider) getType() completionProviderType {
	return keysCompletionType
}

func (p SnippetsProvider) Complete(
	_ context.Context,
	params messages.CompletionParams,
	file files.SimpleObjectFile,
	_ *files.SimpleObjectNode,
	line *yamlastsimple.Line,
) []messages.CompletionItem {
	path := line.GeneralizedPath
	// We only support completions for the root of the document.
	if path != "$" {
		slog.Debug("not a root!", slog.String("path", path), slog.Any("line", line))
		return nil
	}

	prevLine := findLine(file, params.Position.Line-1, filterCommentAndEmptyLines)

	items := make([]messages.CompletionItem, len(p.items))
	copy(items, p.items)
	for i := range p.items {
		if strings.HasPrefix(line.Path, "$[") {
			items[i].InsertText = formatObjectToArrayElement(line, items[i].InsertText)
		} else if prevLine != nil && prevLine.IsType(yamlastsimple.LineTypeList) {
			items[i].InsertText = "- " + formatObjectToArrayElement(line, items[i].InsertText)
		} else if prevLine != nil && !prevLine.IsType(yamlastsimple.LineTypeDocSeparator) {
			// Only add the doc separator if the previous line is not a separator itself.
			items[i].InsertText = "---\n" + items[i].InsertText
		}
	}
	return items
}

func findLine(
	file files.SimpleObjectFile,
	n int,
	filter func(*yamlastsimple.Line) bool,
) *yamlastsimple.Line {
	for _, node := range file {
		for {
			line := node.Doc.FindLine(n - node.Doc.Offset)
			if line == nil {
				break
			}
			if filter(line) {
				n--
				continue
			}
			return line
		}
	}
	return nil
}

func filterCommentAndEmptyLines(line *yamlastsimple.Line) bool {
	return line.IsType(yamlastsimple.LineTypeComment) ||
		line.IsType(yamlastsimple.LineTypeEmpty)
}

type snippetsConfigData struct {
	Name    string `json:"name"`
	Snippet string `json:"snippet"`
}

func setupSnippetItems() []messages.CompletionItem {
	var snippets []snippetsConfigData
	if err := json.Unmarshal(snippetsJSONData, &snippets); err != nil {
		slog.Error("failed to decode snippets", slog.Any("error", err))
		os.Exit(1)
	}
	items := make([]messages.CompletionItem, 0, len(snippets))
	for i := range snippets {
		items = append(items, messages.CompletionItem{
			Label:            snippets[i].Name,
			Kind:             messages.SnippetCompletion,
			InsertText:       snippets[i].Snippet,
			InsertTextFormat: messages.SnippetTextFormat,
		})
	}
	return items
}

// formatObjectToArrayElement formats the raw object as a YAML array element.
func formatObjectToArrayElement(line *yamlastsimple.Line, s string) string {
	switch {
	case line.IsType(yamlastsimple.LineTypeList):
	default:
	}
	lines := strings.Split(s, "\n")
	indented := strings.Builder{}
	indented.Grow(len(s) + 2*len(lines))
	for i := range lines {
		if len(lines[i]) == 0 {
			continue
		}
		if i > 0 {
			indented.WriteString("  ")
		}
		indented.WriteString(lines[i])
		if i < len(lines)-1 {
			indented.WriteByte('\n')
		}
	}
	return indented.String()
}
