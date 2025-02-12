package yamlpath

// PathBuilder represent builder for YAMLPath.
type PathBuilder struct {
	root pathNode
	node pathNode
	path string
}

// Root adds '$' to the current path.
func (b *PathBuilder) Root() *PathBuilder {
	root := newRootNode()
	return &PathBuilder{
		root: root,
		node: root,
	}
}

// Index adds '[<idx>]' to the current path.
func (b *PathBuilder) Index(idx uint) *PathBuilder {
	b.node = b.node.chain(newIndexNode(idx))
	return b
}

// Child adds '.<name>' to the current path.
func (b *PathBuilder) Child(name string) *PathBuilder {
	b.node = b.node.chain(newSelectorNode(name))
	return b
}

// Copy copies the PathBuilder along with its fields.
func (b *PathBuilder) Copy() *PathBuilder {
	root := b.root.copy()
	child := root
	for child != nil {
		if node := child.getChild(); node != nil {
			child = node
			continue
		}
		break
	}
	return &PathBuilder{
		root: root,
		node: child,
		path: b.path,
	}
}

// Build returns the built [Path] pointer.
func (b *PathBuilder) Build() *Path {
	return &Path{node: b.root}
}
