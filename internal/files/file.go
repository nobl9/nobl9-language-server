package files

import (
	"context"
	"strings"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/pkg/errors"

	"github.com/nobl9/nobl9-language-server/internal/yamlast"
	"github.com/nobl9/nobl9-language-server/internal/yamlastsimple"
)

func NewFile(ctx context.Context, uri fileURI, version int, content string) *File {
	file := &File{URI: uri}
	file.Update(ctx, version, content)
	return file
}

// File is a virtual representation of an [os.File].
type File struct {
	URI       fileURI
	Content   string
	Version   int
	Objects   []*ObjectNode
	SimpleAST SimpleObjectFile
	// Err is the error that occurred while parsing the file AST (if any).
	Err error
}

// ObjectNode is a wrapper over single [manifest.Object] which holds both the object and its [yamlast.Node].
type ObjectNode struct {
	Kind    manifest.Kind
	Version manifest.Version
	Object  manifest.Object
	Node    *yamlast.Node
	// Err is the error that occurred while decoding the [manifest.Object] (if any).
	Err error
}

// SimpleObjectFile is a list of [SimpleObjectNode] which are defined in a single file.
type SimpleObjectFile []*SimpleObjectNode

// SimpleObjectNode is a wrapper over single [manifest.Object]
// which holds both the object and its [yamlastsimple.Document].
// It's intended to be used for scenarios where the [ObjectNode] cannot be constructed.
type SimpleObjectNode struct {
	Kind    manifest.Kind
	Version manifest.Version
	Doc     *yamlastsimple.Document
}

// FindObject returns the [ObjectNode] which spans over the specified line.
func (v *File) FindObject(line int) *ObjectNode {
	for _, object := range v.Objects {
		if line >= object.Node.StartLine && line <= object.Node.EndLine {
			return object
		}
	}
	return nil
}

func (v *File) Update(ctx context.Context, version int, content string) {
	// If version has not changed, there's no need to update the file.
	if version == v.Version {
		return
	}
	if version != 0 {
		v.Version = version
	}
	v.Content = content

	v.SimpleAST, v.Err = parseSimpleObjectFile(content)
	if v.Err != nil {
		return
	}

	fileAST, err := yamlast.Parse(content)
	v.Err = err
	if err != nil {
		return
	}
	v.Objects = make([]*ObjectNode, 0, len(fileAST.Nodes))
	for _, node := range fileAST.Nodes {
		object := &ObjectNode{Node: node}
		object.Version, object.Kind, object.Err = inferObjectVersionAndKind(node.Node)
		if object.Err == nil {
			object.Object, object.Err = parseObject(ctx, object)
		}
		v.Objects = append(v.Objects, object)
	}
}

func (v *File) copy() *File {
	objects := make([]*ObjectNode, 0, len(v.Objects))
	for _, object := range v.Objects {
		objects = append(objects, object.copy())
	}
	return &File{
		URI:       v.URI,
		Content:   v.Content,
		Version:   v.Version,
		Objects:   objects,
		SimpleAST: v.SimpleAST,
		Err:       v.Err,
	}
}

func (o *ObjectNode) copy() *ObjectNode {
	return &ObjectNode{
		Kind:    o.Kind,
		Version: o.Version,
		Object:  o.Object,
		Node:    o.Node,
		Err:     o.Err,
	}
}

func parseSimpleObjectFile(content string) (SimpleObjectFile, error) {
	simpleAST := yamlastsimple.ParseFile(content)
	file := make(SimpleObjectFile, 0, len(simpleAST.Docs))
	for _, doc := range simpleAST.Docs {
		switch {
		case len(doc.Lines) == 0:
			continue
		case strings.HasPrefix(doc.Lines[0].Path, "$["):
			docs, err := splitListDocument(doc)
			if err != nil {
				return nil, err
			}
			for _, d := range docs {
				file = append(file, parseSimpleObjectNode(d))
			}
		default:
			file = append(file, parseSimpleObjectNode(doc))
		}
	}
	return file, nil
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
