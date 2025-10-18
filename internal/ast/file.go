package ast

import "strings"

var _ Statement = &File{}

type File struct {
	statement

	Package    *Package
	Statements []Statement
}

func (f *File) Pos() (uint32, uint16) {
	return 0, 0
}

func (f *File) Hash() uint64 {
	return hash(f)
}

func (f *File) String() string {
	var out strings.Builder

	_, _ = out.WriteString(f.Package.String())

	for _, stmt := range f.Statements {
		_, _ = out.WriteString(stmt.String())
		_ = out.WriteByte('\n')
	}

	return out.String()
}
