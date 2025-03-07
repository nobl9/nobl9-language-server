package hover

import (
	_ "embed"
	"strings"
	"sync"
	"text/template"

	"github.com/nobl9/nobl9-go/manifest"

	"github.com/nobl9/nobl9-language-server/internal/sdkdocs"
)

type propertyDocData struct {
	Name     string
	Type     string
	Doc      string
	Rules    []sdkdocs.RulePlan
	Examples []string
}

type objectDocData struct {
	Kind        manifest.Kind
	Name        string
	Description string
	YAML        string
}

//go:embed templates/property-doc.tpl.md
var propertyDocRawTemplate string

//go:embed templates/object-doc.tpl.md
var objectDocRawTemplate string

//go:embed templates/user-doc.tpl.md
var userDocRawTemplate string

var (
	propertyDocTpl = lazyGetTemplate(propertyDocRawTemplate)
	objectDocTpl   = lazyGetTemplate(objectDocRawTemplate)
	userDocTpl     = lazyGetTemplate(userDocRawTemplate)
)

func lazyGetTemplate(raw string) func() *template.Template {
	var (
		tpl  *template.Template
		once sync.Once
	)
	return func() *template.Template {
		once.Do(func() {
			tpl = template.Must(template.New("").
				Funcs(template.FuncMap{
					"markdownEscape": markdownEscape,
				}).
				Parse(raw))
		})
		return tpl
	}
}

// markdownEscape escapes markdown characters in the given string.
func markdownEscape(s string) string {
	return markdownReplacer.Replace(s)
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
