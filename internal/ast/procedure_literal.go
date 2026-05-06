package ast

import (
	"fmt"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &ProcedureLiteral{}

type ProcedureLiteral struct {
	Token         tokens.Token
	ProcedureType types.Type
	Body          *Block
}

func (a *AST) NewProcedureLiteral(token tokens.Token, t *types.Procedure, body *Block) ExprIndex {
	procLiteral := New[ProcedureLiteral](a)
	procLiteral.Token = token
	procLiteral.ProcedureType = t
	procLiteral.Body = body
	return a.AddExpr(procLiteral)
}

func (e *ProcedureLiteral) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *ProcedureLiteral) Hash() uint64 {
	return hash(e)
}

func (e *ProcedureLiteral) StringTo(out *strings.Builder, a *AST) {
	_ = out.WriteByte('{')

	for i, s := range e.Body.Statements {
		if i == 0 {
			_ = out.WriteByte('\n')
		}

		stmt := a.nodes[s]

		ln, col := stmt.Pos()
		fmt.Fprintf(out, "ln %d, col %d:", ln, col)

		_ = out.WriteByte('\t')
		stmt.StringTo(out, a)
		_ = out.WriteByte('\n')
	}

	_ = out.WriteByte('}')
}

func (e *ProcedureLiteral) Type() types.Type {
	return e.ProcedureType
}
