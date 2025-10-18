package ast

import "github.com/samborkent/cog/internal/tokens"

var _ Statement = &ExpressionStatement{}

type ExpressionStatement struct {
	statement

	Token      tokens.Token
	Expression Expression
}

func (s *ExpressionStatement) Hash() uint64 {
	return hash(s)
}

func (s *ExpressionStatement) Pos() (uint32, uint16) {
	return s.Token.Ln, s.Token.Col
}

func (s *ExpressionStatement) String() string {
	return s.Expression.String()
}
