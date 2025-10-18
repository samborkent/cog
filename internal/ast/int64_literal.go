package ast

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	"strconv"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Int64Literal{}

type Int64Literal struct {
	expression

	Token tokens.Token
	Value int64
}

func NewInt64Literal(t tokens.Token) (*Int64Literal, error) {
	value, err := strconv.ParseInt(t.Literal, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse int literal: %w", err)
	}

	return &Int64Literal{
		Token: t,
		Value: value,
	}, nil
}

func (l *Int64Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Int64Literal) Go() *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.INT,
		Value: strconv.FormatInt(l.Value, 10),
	}
}

func (l *Int64Literal) Hash() uint64 {
	return hash(l)
}

func (l *Int64Literal) String() string {
	return "(" + strconv.FormatInt(l.Value, 10) + " : int64)"
}

func (l *Int64Literal) Type() types.Type {
	return types.Basics[types.Int64]
}
