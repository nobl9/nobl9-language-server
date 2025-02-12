package server

import (
	"context"
	"strings"

	"github.com/nobl9/nobl9-go/manifest"

	"github.com/nobl9/nobl9-language-server/internal/yamlast"
	"github.com/nobl9/nobl9-language-server/internal/yamlastfast"
)

func newVirtualFile(ctx context.Context, uri fileURI, content string) (*virtualFile, error) {
	file := &virtualFile{
		URI:     uri,
		Content: content,
	}
	if err := file.Update(ctx, content); err != nil {
		return nil, err
	}
	return file, nil
}

// virtualFile is a virtual representation of an [os.File].
type virtualFile struct {
	URI     fileURI
	Content string
	Version int
	Objects []*objectNode
	FastAST []*fastObjectNode
	// Err is the error that occurred while parsing the file AST (if any).
	Err error
}

// objectNode is a wrapper over single [manifest.Object] which holds both the object and its [yamlast.Node].
type objectNode struct {
	Kind    manifest.Kind
	Version manifest.Version
	Object  manifest.Object
	Node    *yamlast.Node
	// Err is the error that occurred while decoding the [manifest.Object] (if any).
	Err error
}

// fastObjectNode is a wrapper over single [manifest.Object] which holds both the object and its [yamlastfast.Document].
type fastObjectNode struct {
	Kind    manifest.Kind
	Version manifest.Version
	Doc     *yamlastfast.Document
}

func (o *objectNode) copy() *objectNode {
	return &objectNode{
		Kind:    o.Kind,
		Version: o.Version,
		Object:  o.Object,
		Node:    o.Node,
		Err:     o.Err,
	}
}

// FindObject returns the [objectNode] which spans over the specified line.
func (v *virtualFile) FindObject(line int) *objectNode {
	for _, object := range v.Objects {
		if line >= object.Node.StartLine && line <= object.Node.EndLine {
			return object
		}
	}
	return nil
}

func (v *virtualFile) UpdateVersion(version int) {
	v.Version = version
}

func (v *virtualFile) Update(ctx context.Context, content string) error {
	v.Content = content

	fastAST := yamlastfast.ParseFile(content)
	v.FastAST = make([]*fastObjectNode, 0, len(fastAST.Docs))
	for _, doc := range fastAST.Docs {
		object := &fastObjectNode{Doc: doc}
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
		v.FastAST = append(v.FastAST, object)
	}

	fileAST, err := yamlast.Parse(content)
	v.Err = err
	if err != nil {
		return nil // nolint: nilerr
	}
	v.Objects = make([]*objectNode, 0, len(fileAST.Nodes))
	for _, node := range fileAST.Nodes {
		object := &objectNode{Node: node}
		object.Version, object.Kind, object.Err = inferObjectVersionAndKind(node.Node)
		if object.Err == nil {
			object.Object, object.Err = ParseObject(ctx, object)
		}
		v.Objects = append(v.Objects, object)
	}
	return nil
}

func (v *virtualFile) copy() *virtualFile {
	objects := make([]*objectNode, 0, len(v.Objects))
	for _, object := range v.Objects {
		objects = append(objects, object.copy())
	}
	return &virtualFile{
		URI:     v.URI,
		Content: v.Content,
		Version: v.Version,
		Objects: objects,
		FastAST: v.FastAST,
		Err:     v.Err,
	}
}
