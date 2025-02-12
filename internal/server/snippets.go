package server

import (
	_ "embed"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/yamlastfast"
	"github.com/nobl9/nobl9-language-server/internal/yamlpath"
)

//go:embed snippets.json
var snippetsJSONData []byte

func newSnippetsProvider() *snippetsProvider {
	return &snippetsProvider{items: setupSnippetItems()}
}

type snippetsProvider struct {
	items []messages.CompletionItem
}

func (s snippetsProvider) Complete(
	docs []*fastObjectNode,
	lineNum int,
	line *yamlastfast.Line,
) []messages.CompletionItem {
	findLine := func(n int) *yamlastfast.Line {
		for _, doc := range docs {
			line := doc.Doc.FindLine(n - doc.Doc.Offset)
			if line != nil {
				return line
			}
		}
		return nil
	}

	path := line.Path
	// We only support completions for the root of the document.
	isRootArray := yamlpath.Match("$[*]", path)
	if path != "$" && !isRootArray {
		slog.Debug("not a root!", slog.String("path", path), slog.Any("line", line))
		return nil
	}

	items := make([]messages.CompletionItem, len(s.items))
	copy(items, s.items)
	for i := range s.items {
		prevLine := findLine(lineNum - 1)
		if isRootArray {
			items[i].InsertText = formatObjectToArrayElement(line, items[i].InsertText)
		} else if prevLine != nil && prevLine.IsType(yamlastfast.LineTypeList) {
			items[i].InsertText = "- " + formatObjectToArrayElement(line, items[i].InsertText)
		} else if prevLine != nil && !prevLine.IsType(yamlastfast.LineTypeDocSeparator) {
			// Only add the doc separator if the previous line is not a separator itself.
			items[i].InsertText = "---\n" + items[i].InsertText
		}
	}
	return items
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
func formatObjectToArrayElement(line *yamlastfast.Line, s string) string {
	switch {
	case line.IsType(yamlastfast.LineTypeList):
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
