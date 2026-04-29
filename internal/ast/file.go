package ast

import "strings"

var _ Node = &File{}

type File struct {
	Name         string
	Package      *Package
	Statements   []NodeIndex
	ContainsMain bool
}

func (a *AST) NewFile(name string, pkg *Package, statements []NodeIndex, containsMain bool) NodeIndex {
	node := New[File](a)
	node.Name = name
	node.Package = pkg
	node.Statements = statements
	node.ContainsMain = containsMain
	return a.AddNode(node)
}

func (n *File) Pos() (uint32, uint16) {
	return 0, 0
}

func (n *File) Hash() uint64 {
	return hash(n)
}

func (n *File) StringTo(out *strings.Builder, a *AST) {
	_, _ = out.WriteString(n.Name)
	_ = out.WriteByte('\n')

	n.Package.StringTo(out, a)

	for _, stmt := range n.Statements {
		a.nodes[stmt].StringTo(out, a)
		_ = out.WriteByte('\n')
	}
}

func (n *File) String() string {
	var out strings.Builder
	n.StringTo(&out, nil)
	return out.String()
}
