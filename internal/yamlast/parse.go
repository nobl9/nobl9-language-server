package yamlast

import (
	"slices"
	"strings"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/lexer"
	"github.com/goccy/go-yaml/parser"
)

type File struct {
	Nodes []*Node
	File  *ast.File
	Err   error
}

// Parse parses the content of a YAML file and returns its AST represented through a flat list of [Node].
// This means top level sequence nodes are flattened into individual documents.
// Each [Node] contains the [ast.Node] and the start and end line of the node in the original file.
func Parse(content string) (*File, error) {
	tokens := lexer.Tokenize(content)
	astFile, err := parser.Parse(tokens, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	nodes := make([]*Node, 0, len(astFile.Docs))
	for _, doc := range astFile.Docs {
		if doc.Start != nil {
			// Add a dummy node for the document separator.
			nodes = append(nodes, &Node{
				StartLine:   doc.Start.Position.Line,
				EndLine:     doc.Start.Position.Line,
				isSeparator: true,
			})
		}
		switch nv := doc.Body.(type) {
		case *ast.SequenceNode:
			for _, node := range nv.Values {
				nodes = append(nodes, &Node{
					StartLine: node.GetToken().Position.Line,
					Node:      node,
				})
			}
		default:
			node := &Node{Node: doc.Body}
			switch {
			case doc.Body != nil:
				node.StartLine = doc.Body.GetToken().Position.Line
			case doc.Start != nil:
				node.StartLine = doc.Start.Position.Line
			}
			nodes = append(nodes, node)
		}
	}
	for i, doc := range nodes {
		if i+1 < len(nodes) {
			doc.EndLine = nodes[i+1].StartLine - 1
		} else {
			doc.EndLine = strings.Count(content, "\n") + 1
		}
	}
	// Filter out dummy separator nodes.
	nodes = slices.DeleteFunc(nodes, func(n *Node) bool { return n.isSeparator })
	return &File{
		Nodes: nodes,
		File:  astFile,
	}, nil
}
