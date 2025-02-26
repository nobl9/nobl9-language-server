package sdkdocs

import (
	_ "embed"
	"encoding/json"

	"github.com/nobl9/nobl9-go/manifest"

	"github.com/nobl9/nobl9-language-server/internal/yamlpath"
)

//go:embed docs.json
var docsJSON []byte

func New() (*Docs, error) {
	var docs []ObjectDoc
	if err := json.Unmarshal(docsJSON, &docs); err != nil {
		return nil, err
	}
	docsMap := make(map[manifest.Kind][]*PropertyDoc, len(docs))
	deprecatedMap := make(map[manifest.Kind][]string)
	for _, doc := range docs {
		for _, property := range doc.Properties {
			docsMap[doc.Kind] = append(docsMap[doc.Kind], &property)
		}
		deprecatedMap[doc.Kind] = selectDeprecatedPaths(docsMap[doc.Kind])
	}
	return &Docs{
		m:          docsMap,
		deprecated: deprecatedMap,
	}, nil
}

type Docs struct {
	m          map[manifest.Kind][]*PropertyDoc
	deprecated map[manifest.Kind][]string
}

// GetProperty returns a [PropertyDoc] matching provided kind and path.
// If the property was not found it returns nil.
func (s Docs) GetProperty(kind manifest.Kind, path string) *PropertyDoc {
	docs, ok := s.m[kind]
	if !ok {
		return nil
	}
	for _, doc := range docs {
		if yamlpath.Match(doc.Path, path) {
			return doc
		}
	}
	return nil
}

// GetDeprecatedPaths returns a list of deprecated paths for the given kind.
func (s Docs) GetDeprecatedPaths(kind manifest.Kind) []string {
	return s.deprecated[kind]
}

func selectDeprecatedPaths(docs []*PropertyDoc) []string {
	deprecated := make([]string, 0, len(docs))
	for _, doc := range docs {
		if doc.IsDeprecated {
			deprecated = append(deprecated, doc.Path)
		}
	}
	return deprecated
}
