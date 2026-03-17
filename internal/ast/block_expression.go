package ast

import (
	"fmt"
	"strings"

	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &ProcedureLiteral{}

type ProcedureLiteral struct {
	expression

	Body          *Block
	ProcedureType types.Type
}

func (l *ProcedureLiteral) Pos() (uint32, uint16) {
	return l.Body.Start.Ln, l.Body.Start.Col
}

func (l *ProcedureLiteral) Hash() uint64 {
	return hash(l)
}

func (l *ProcedureLiteral) stringTo(out *strings.Builder) {
	_ = out.WriteByte('{')

	for i, stmt := range l.Body.Statements {
		if i == 0 {
			_ = out.WriteByte('\n')
		}

		ln, col := stmt.Pos()
		fmt.Fprintf(out, "ln %d, col %d:", ln, col)

		_ = out.WriteByte('\t')
		stmt.stringTo(out)
		_ = out.WriteByte('\n')
	}

	_ = out.WriteByte('}')
}

func (l *ProcedureLiteral) String() string {
	var out strings.Builder
	l.stringTo(&out)
	return out.String()
}

func (l *ProcedureLiteral) Type() types.Type {
	if l.ProcedureType == nil {
		panic("procedure literal with nil type detected")
	}

	return l.ProcedureType
}
