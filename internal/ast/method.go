package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Node = &Method{}

type Method struct {
	Token         tokens.Token
	Export        bool
	ReceiverIdent *Identifier // may be nil
	ReceiverType  types.Type
	Type          *types.Procedure
	Body          ExprIndex // ProcedureLiteral
}

func (a *AST) NewMethod(
	t tokens.Token, export bool, receiverIdent *Identifier,
	receiverType types.Type, typ *types.Procedure, body ExprIndex,
) NodeIndex {
	node := New[Method](a)

	node.Token = t
	node.Export = export
	node.ReceiverIdent = receiverIdent
	node.ReceiverType = receiverType
	node.Type = typ
	node.Body = body

	return a.AddNode(node)
}

func (n *Method) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Method) Hash() uint64 {
	return hash(n)
}

func (n *Method) StringTo(out *strings.Builder, a *AST) {
	if n.Export {
		_, _ = out.WriteString("export ")
	}

	if n.ReceiverIdent != nil {
		_ = out.WriteByte('(')
		_, _ = out.WriteString(n.ReceiverIdent.Token.Literal)
		_, _ = out.WriteString(" : ")
		_, _ = out.WriteString(n.Type.String())
		_ = out.WriteByte(')')
	} else {
		_, _ = out.WriteString(n.Type.String())
	}

	_ = out.WriteByte('.')
	_, _ = out.WriteString(n.Token.Literal)
	_, _ = out.WriteString(" : ")
	_, _ = out.WriteString(n.Type.String())

	_, _ = out.WriteString(" = ")

	a.exprs[n.Body].StringTo(out, a)
}
