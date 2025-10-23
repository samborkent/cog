package ast

import (
	"github.com/samborkent/cog/internal/types"
)

var _ Statement = &Declaration{}

type Declaration struct {
	statement

	Assignment *Assignment
	Constant   bool
}

func (d *Declaration) Pos() (uint32, uint16) {
	return d.Assignment.Token.Ln, d.Assignment.Token.Col
}

func (d *Declaration) Hash() uint64 {
	return hash(d)
}

func (d *Declaration) String() string {
	var prefix string

	if d.Assignment.Identifier.Exported {
		prefix = "export "
	}

	if d.Constant {
		prefix = prefix + "const "
	}

	if d.Assignment.Expression == nil {
		return prefix + d.Assignment.Identifier.String() + " : " + d.Assignment.Identifier.ValueType.String()
	}

	if d.Assignment.Identifier.ValueType == nil || d.Assignment.Identifier.ValueType == types.None {
		return prefix + d.Assignment.Identifier.String() + " := " + d.Assignment.Expression.String()
	}

	return prefix + d.Assignment.Identifier.String() + " : " + d.Assignment.Identifier.ValueType.String() + " = " + d.Assignment.Expression.String()
}
