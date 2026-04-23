package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &ExpressionStatement{}

type ExpressionStatement struct {
	Token tokens.Token
	Expr  ExprValue
}

func (s *ExpressionStatement) Kind() NodeKind {
	return KindExpressionStatement
}

func (s *ExpressionStatement) Hash() uint64 {
	return hash(s)
}

func (s *ExpressionStatement) Pos() (uint32, uint16) {
	return s.Token.Ln, s.Token.Col
}

func (s *ExpressionStatement) stringTo(out *strings.Builder) {
	s.Expr.expr.stringTo(out)
}

func (s *ExpressionStatement) String() string {
	var out strings.Builder
	s.stringTo(&out)

	return out.String()
}
