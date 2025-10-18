package ast

import "github.com/samborkent/cog/internal/tokens"

var _ Statement = &Assignment{}

type Assignment struct {
	statement

	Token      tokens.Token
	Identifier *Identifier
	Expression Expression
}

func (a *Assignment) Pos() (uint32, uint16) {
	return a.Token.Ln, a.Token.Col
}

func (a *Assignment) Hash() uint64 {
	return hash(a)
}

func (a *Assignment) String() string {
	return a.Identifier.String() + " " + a.Token.Type.String() + " " + a.Expression.String()
}
