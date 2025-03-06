package yamlast

import (
	"github.com/goccy/go-yaml/ast"
)

// Node is a wrapper over [ast.Node] that also contains the start and end line of the node in the original file.
// Both StartLine and EndLine are inclusive range endpoints.
type Node struct {
	Node        ast.Node
	StartLine   int
	EndLine     int
	isSeparator bool
}

// Find finds an [ast.Node] in the [Node] that's located at the specified line and character.
// Line number must be an absolute line number in the whole file.
func (n Node) Find(line int) (ast.Node, error) {
	finder := &nodeFinder{line: line, Node: n.Node}
	finder.walk(n.Node)
	return finder.Node, nil
}

func (n Node) Copy() *Node {
	return &Node{
		Node:      n.Node,
		StartLine: n.StartLine,
		EndLine:   n.EndLine,
	}
}

type nodeFinder struct {
	Node ast.Node
	line int
}

func (n *nodeFinder) walk(node ast.Node) {
	switch nv := node.(type) {
	case *ast.MappingNode:
		n.walk(findClosestNode(nv.Values, n.line))
	case *ast.MappingValueNode:
		// We found the exact node.
		if nv.Start.Position.Line == n.line {
			n.Node = node
			return
		}
		// We stepped over, the previous node was the closest.
		if nv.Start.Position.Line > n.line {
			return
		}
		n.Node = node
		n.walk(nv.Key)
		n.walk(nv.Value)
	case *ast.SequenceNode:
		n.walk(findClosestNode(nv.Values, n.line))
	}
}

// findClosestNode finds the closest [ast.Node] to the given line in slice of [ast.Node].
func findClosestNode[T ast.Node](nodes []T, line int) (match ast.Node) {
	lineCandidate := 0
	for _, node := range nodes {
		token := node.GetToken()
		if token == nil || token.Position == nil {
			continue
		}
		if token.Position.Line == line {
			return node
		}
		if token.Position.Line < line && token.Position.Line > lineCandidate {
			match = node
			lineCandidate = token.Position.Line
		}
	}
	return match
}
