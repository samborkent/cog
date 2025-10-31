package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &ProcedureLiteral{}

type ProcedureLiteral struct {
	expression

	Body         *Block
	ProcdureType types.Type
}

func (l *ProcedureLiteral) Pos() (uint32, uint16) {
	return l.Body.Start.Ln, l.Body.Start.Col
}

func (l *ProcedureLiteral) Hash() uint64 {
	return hash(l)
}

func (l *ProcedureLiteral) String() string {
	var out strings.Builder

	_ = out.WriteByte('{')

	for i, stmt := range l.Body.Statements {
		if i == 0 {
			_ = out.WriteByte('\n')
		}

		_ = out.WriteByte('\t')
		_, _ = out.WriteString(stmt.String())
		_ = out.WriteByte('\n')
	}

	_ = out.WriteByte('}')

	return out.String()
}

func (l *ProcedureLiteral) Type() types.Type {
	if l.ProcdureType == nil {
		panic("procedure literal with nil type detected")
	}

	return l.ProcdureType
}
