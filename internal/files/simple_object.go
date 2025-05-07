package files

import (
	"strconv"
	"strings"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/pkg/errors"

	"github.com/nobl9/nobl9-language-server/internal/yamlastsimple"
)

// SimpleObjectFile is a list of [SimpleObjectNode] which are defined in a single file.
type SimpleObjectFile []*SimpleObjectNode

// SimpleObjectNode is a wrapper over single [manifest.Object]
// which holds both the object and its [yamlastsimple.Document].
// It's intended to be used for scenarios where the [ObjectNode] cannot be constructed.
type SimpleObjectNode struct {
	Kind    manifest.Kind
	Version manifest.Version
	Doc     *yamlastsimple.Document
	// isListElement is true if the object node is a list element.
	isListElement    bool
	listElementIndex int
}

// FindLineByPath returns the [yamlastsimple.Line] which has the specified path.
// If the [SimpleObjectNode] is a list element and path does not have list prefix
// the appropriate list index will be prepended.
func (s SimpleObjectNode) FindLineByPath(path string) *yamlastsimple.Line {
	if s.isListElement && !strings.HasPrefix(path, "$[") {
		if len(path) > 0 {
			path = path[:0] + "$[" + strconv.Itoa(s.listElementIndex) + "]" + path[1:]
		}
	}
	for _, line := range s.Doc.Lines {
		if line.Path == path {
			return line
		}
	}
	return nil
}

func ParseSimpleObjectFile(content string) (SimpleObjectFile, error) {
	simpleAST := yamlastsimple.ParseFile(content)
	file := make(SimpleObjectFile, 0, len(simpleAST.Docs))
	for _, doc := range simpleAST.Docs {
		switch {
		case len(doc.Lines) == 0:
			continue
		case docIsList(doc):
			docs, err := splitListDocument(doc)
			if err != nil {
				return nil, err
			}
			for i, d := range docs {
				node := parseSimpleObjectNode(d)
				node.isListElement = true
				node.listElementIndex = i
				file = append(file, node)
			}
		default:
			node := parseSimpleObjectNode(doc)
			file = append(file, node)
		}
	}
	return file, nil
}

func docIsList(doc *yamlastsimple.Document) bool {
	for _, line := range doc.Lines {
		if line.IsType(yamlastsimple.LineTypeList) &&
			strings.HasPrefix(line.Path, "$[") {
			return true
		}
	}
	return false
}

func parseSimpleObjectNode(doc *yamlastsimple.Document) *SimpleObjectNode {
	object := &SimpleObjectNode{Doc: doc}
	for _, line := range doc.Lines {
		dotIndex := strings.Index(line.Path, ".")
		if dotIndex == -1 {
			continue
		}
		if line.Path[dotIndex+1:] == "kind" {
			object.Kind, _ = manifest.ParseKind(line.GetMapValue())
		}
		if line.Path[dotIndex+1:] == "apiVersion" {
			object.Version, _ = manifest.ParseVersion(line.GetMapValue())
		}
	}
	return object
}

func splitListDocument(doc *yamlastsimple.Document) ([]*yamlastsimple.Document, error) {
	var listPrefix string
	result := make([]*yamlastsimple.Document, 0)
	currentDoc := &yamlastsimple.Document{Offset: doc.Offset}
	for _, line := range doc.Lines {
		// At least $[\d+] is expected, otherwise we might be dealing with an empty line.
		if len(line.Path) < 4 {
			currentDoc.Lines = append(currentDoc.Lines, line)
			continue
		}
		closingBracketIdx := strings.Index(line.Path, "]")
		if closingBracketIdx == -1 {
			return nil, errors.New("invalid list index (missing closing bracket): " + line.Path)
		}
		newListPrefix := line.Path[2:closingBracketIdx]
		// Initial list element.
		if listPrefix == "" {
			listPrefix = newListPrefix
		}
		if listPrefix != newListPrefix {
			// Remove list prefix from path.
			listPrefix = newListPrefix
			result = append(result, currentDoc)
			currentDoc = &yamlastsimple.Document{Offset: currentDoc.Offset + len(currentDoc.Lines)}
		}
		currentDoc.Lines = append(currentDoc.Lines, line)
	}
	return append(result, currentDoc), nil
}
