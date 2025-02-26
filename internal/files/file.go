package files

import (
	"context"
	"strings"

	"github.com/nobl9/nobl9-go/manifest"

	"github.com/nobl9/nobl9-language-server/internal/yamlast"
	"github.com/nobl9/nobl9-language-server/internal/yamlastsimple"
)

func NewFile(ctx context.Context, uri fileURI, content string) (*File, error) {
	file := &File{
		URI:     uri,
		Content: content,
	}
	if err := file.Update(ctx, content); err != nil {
		return nil, err
	}
	return file, nil
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

func (v *File) UpdateVersion(version int) {
	v.Version = version
}

func (v *File) Update(ctx context.Context, content string) error {
	v.Content = content

	simpleAST := yamlastsimple.ParseFile(content)
	v.SimpleAST = make([]*SimpleObjectNode, 0, len(simpleAST.Docs))
	for _, doc := range simpleAST.Docs {
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
		v.SimpleAST = append(v.SimpleAST, object)
	}

	fileAST, err := yamlast.Parse(content)
	v.Err = err
	if err != nil {
		return nil // nolint: nilerr
	}
	v.Objects = make([]*ObjectNode, 0, len(fileAST.Nodes))
	for _, node := range fileAST.Nodes {
		object := &ObjectNode{Node: node}
		object.Version, object.Kind, object.Err = inferObjectVersionAndKind(node.Node)
		if object.Err == nil {
			object.Object, object.Err = ParseObject(ctx, object)
		}
		v.Objects = append(v.Objects, object)
	}
	return nil
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
