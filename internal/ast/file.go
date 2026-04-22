package ast

import "strings"

var _ Node = &File{}

type File struct {
	Name         string
	Package      *Package
	Statements   []NodeValue
	ContainsMain bool
}

func (f *File) Pos() (uint32, uint16) {
	return 0, 0
}

func (f *File) Hash() uint64 {
	return hash(f)
}

func (f *File) stringTo(out *strings.Builder) {
	_, _ = out.WriteString(f.Name)
	_ = out.WriteByte('\n')

	f.Package.stringTo(out)

	for _, stmt := range f.Statements {
		stmt.node.stringTo(out)
		_ = out.WriteByte('\n')
	}
}

func (f *File) String() string {
	var out strings.Builder
	f.stringTo(&out)

	return out.String()
}
