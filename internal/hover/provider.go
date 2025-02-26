package hover

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/nobl9/nobl9-go/manifest"

	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/sdkdocs"
	"github.com/nobl9/nobl9-language-server/internal/yamlpath"
)

type docsProvider interface {
	GetProperty(kind manifest.Kind, path string) *sdkdocs.PropertyDoc
}

func NewProvider(docs docsProvider) *Provider {
	return &Provider{docs: docs}
}

type Provider struct {
	docs docsProvider
}

func (d Provider) Hover(kind manifest.Kind, path string) *messages.HoverResponse {
	path = yamlpath.NormalizeRootPath(path)
	prop := d.docs.GetProperty(kind, path)
	if prop == nil {
		return nil
	}
	docs := d.buildDocs(prop)
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

func (d Provider) buildDocs(doc *sdkdocs.PropertyDoc) string {
	b := strings.Builder{}
	lastDot := strings.LastIndex(doc.Path, ".")
	propertyName := doc.Path[lastDot+1:]
	b.WriteString(fmt.Sprintf("`%s:%s`\n\n", propertyName, doc.Type))
	b.WriteString(doc.Doc)
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
