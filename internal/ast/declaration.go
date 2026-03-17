package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/types"
)

var _ Statement = &Declaration{}

type Declaration struct {
	statement

	Assignment *Assignment
}

func (d *Declaration) Pos() (uint32, uint16) {
	return d.Assignment.Token.Ln, d.Assignment.Token.Col
}

func (d *Declaration) Hash() uint64 {
	return hash(d)
}

func (d *Declaration) stringTo(out *strings.Builder) {
	if d.Assignment.Identifier.Exported {
		_, _ = out.WriteString("export ")
	}

	switch d.Assignment.Identifier.Qualifier {
	case QualifierVariable:
		_, _ = out.WriteString("var ")
	case QualifierDynamic:
		_, _ = out.WriteString("dyn ")
	}

	if d.Assignment.Expression == nil {
		d.Assignment.Identifier.stringTo(out)
		_, _ = out.WriteString(" : ")
		_, _ = out.WriteString(d.Assignment.Identifier.ValueType.String())

		return
	}

	if d.Assignment.Identifier.ValueType == nil || d.Assignment.Identifier.ValueType == types.None {
		d.Assignment.Identifier.stringTo(out)
		_, _ = out.WriteString(" := ")
		d.Assignment.Expression.stringTo(out)

		return
	}

	d.Assignment.Identifier.stringTo(out)
	_, _ = out.WriteString(" : ")
	_, _ = out.WriteString(d.Assignment.Identifier.ValueType.String())
	_, _ = out.WriteString(" = ")
	d.Assignment.Expression.stringTo(out)
}

func (d *Declaration) String() string {
	var out strings.Builder
	d.stringTo(&out)
	return out.String()
}
