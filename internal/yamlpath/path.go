package yamlpath

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/pkg/errors"
)

// Path represents a single YAMLPath (like).
type Path struct {
	path string
	node pathNode
}

// String returns textual Path representation.
func (p *Path) String() string {
	return p.node.String()
}

// FilterFile filters the whole [ast.File].
func (p *Path) FilterFile(f *ast.File) (ast.Node, bool, error) {
	for _, doc := range f.Docs {
		node, firstMatch, err := p.FilterNode(doc.Body)
		if err != nil {
			return nil, firstMatch, errors.Wrapf(err, "failed to filter node by path: %s ", p.path)
		}
		if node != nil {
			return node, firstMatch, nil
		}
	}
	return nil, false, errors.Wrapf(yaml.ErrNotFoundNode, "failed to find path: %s", p.path)
}

// FilterNode filters single [ast.Node].
// If it cannot find the filtered [ast.Node], it will remove the last element
// from the path and repeat this step until a match is found.
// In the end, if none of the nodes matched the selector, a root node will be returned.
func (p *Path) FilterNode(node ast.Node) (ast.Node, bool, error) {
	firstMatch := true
	for {
		n, err := p.node.filter(node)
		if err != nil {
			return nil, firstMatch, errors.Wrapf(err, "failed to filter node by path: %s", p.path)
		}
		if n != nil {
			return n, firstMatch, nil
		}
		p.node.removeChild()
		firstMatch = false
	}
}

type pathNode interface {
	fmt.Stringer
	chain(pathNode) pathNode
	filter(ast.Node) (ast.Node, error)
	getChild() pathNode
	removeChild()
	copy() pathNode
}

type rootNode struct {
	child pathNode
}

func newRootNode() *rootNode {
	return &rootNode{}
}

func (n *rootNode) String() string {
	s := "$"
	if n.child != nil {
		s += n.child.String()
	}
	return s
}

func (n *rootNode) chain(node pathNode) pathNode {
	n.child = node
	return node
}

func (n *rootNode) filter(node ast.Node) (ast.Node, error) {
	if n.child == nil {
		return node, nil
	}
	filtered, err := n.child.filter(node)
	if err != nil {
		return nil, errors.Wrap(err, "failed to filter")
	}
	return filtered, nil
}

func (n *rootNode) getChild() pathNode {
	return n.child
}

func (n *rootNode) removeChild() {
	if n.child == nil {
		return
	}
	if n.child.getChild() == nil {
		n.child = nil
		return
	}
	n.child.removeChild()
}

func (n *rootNode) copy() pathNode {
	return &rootNode{child: copyChildNode(n.child)}
}

type selectorNode struct {
	child    pathNode
	selector string
}

func newSelectorNode(selector string) *selectorNode {
	return &selectorNode{selector: selector}
}

func (n *selectorNode) String() string {
	selector := n.normalizeSelectorName(n.selector)
	s := fmt.Sprintf(".%s", selector)
	if n.child != nil {
		s += n.child.String()
	}
	return s
}

func (n *selectorNode) normalizeSelectorName(name string) string {
	if strings.HasPrefix(name, "'") && strings.HasSuffix(name, "'") {
		// Name was already escaped.
		return name
	}
	if strings.Contains(name, ".") || strings.Contains(name, "*") {
		escapedName := strings.ReplaceAll(name, `'`, `\'`)
		return "'" + escapedName + "'"
	}
	return name
}

func (n *selectorNode) chain(node pathNode) pathNode {
	n.child = node
	return node
}

func (n *selectorNode) filter(node ast.Node) (ast.Node, error) {
	selector := n.selector
	if len(selector) > 1 && selector[0] == '\'' && selector[len(selector)-1] == '\'' {
		selector = selector[1 : len(selector)-1]
	}
	switch node.Type() {
	case ast.MappingType:
		return n.filterMappingTypeNode(node.(*ast.MappingNode), selector)
	case ast.MappingValueType:
		return n.filterMappingValueNode(node.(*ast.MappingValueNode), selector)
	default:
		return nil, errors.Wrapf(yaml.ErrInvalidQuery, "expected node type is map or map value. but got %s", node.Type())
	}
}

func (n *selectorNode) filterMappingTypeNode(node *ast.MappingNode, selector string) (ast.Node, error) {
	var err error
	for _, value := range node.Values {
		key := value.Key.GetToken().Value
		if len(key) > 0 {
			key, err = unquoteKey(key)
			if err != nil {
				return nil, err
			}
		}
		if key != selector {
			continue
		}
		if n.child == nil {
			return value, nil
		}
		filtered, err := n.child.filter(value.Value)
		if err != nil {
			return nil, errors.Wrap(err, "failed to filter")
		}
		return filtered, nil
	}
	return nil, nil
}

func (n *selectorNode) filterMappingValueNode(node *ast.MappingValueNode, selector string) (ast.Node, error) {
	key := node.Key.GetToken().Value
	if key != selector {
		return nil, nil
	}
	if n.child == nil {
		return node, nil
	}
	filtered, err := n.child.filter(node.Value)
	if err != nil {
		return nil, errors.Wrap(err, "failed to filter")
	}
	return filtered, nil
}

func (n *selectorNode) getChild() pathNode {
	return n.child
}

func (n *selectorNode) removeChild() {
	if n.child == nil {
		return
	}
	if n.child.getChild() == nil {
		n.child = nil
		return
	}
	n.child.removeChild()
}

func (n *selectorNode) copy() pathNode {
	return &selectorNode{selector: n.selector, child: copyChildNode(n.child)}
}

type indexNode struct {
	child    pathNode
	selector uint
}

func newIndexNode(selector uint) *indexNode {
	return &indexNode{selector: selector}
}

func (n *indexNode) String() string {
	s := fmt.Sprintf("[%d]", n.selector)
	if n.child != nil {
		s += n.child.String()
	}
	return s
}

func (n *indexNode) chain(node pathNode) pathNode {
	n.child = node
	return node
}

func (n *indexNode) filter(node ast.Node) (ast.Node, error) {
	if node.Type() != ast.SequenceType {
		return nil, errors.Wrapf(yaml.ErrInvalidQuery, "expected sequence type node. but got %s", node.Type())
	}
	sequence := node.(*ast.SequenceNode)
	if n.selector >= uint(len(sequence.Values)) {
		return nil, errors.Wrapf(
			yaml.ErrInvalidQuery,
			"expected index is %d. but got sequences has %d items",
			n.selector,
			sequence.Values,
		)
	}
	value := sequence.Values[n.selector]
	if n.child == nil {
		return value, nil
	}
	filtered, err := n.child.filter(value)
	if err != nil {
		return nil, errors.Wrap(err, "failed to filter")
	}
	return filtered, nil
}

func (n *indexNode) getChild() pathNode {
	return n.child
}

func (n *indexNode) removeChild() {
	if n.child == nil {
		return
	}
	if n.child.getChild() == nil {
		n.child = nil
		return
	}
	n.child.removeChild()
}

func (n *indexNode) copy() pathNode {
	return &indexNode{selector: n.selector, child: copyChildNode(n.child)}
}

func unquoteKey(key string) (string, error) {
	switch key[0] {
	case '"':
		var err error
		key, err = strconv.Unquote(key)
		if err != nil {
			return "", errors.Wrap(err, "failed to unquote key")
		}
	case '\'':
		if len(key) > 1 && key[len(key)-1] == '\'' {
			key = key[1 : len(key)-1]
		}
	}
	return key, nil
}

func copyChildNode(n pathNode) pathNode {
	if n == nil {
		return n
	}
	return n.copy()
}
