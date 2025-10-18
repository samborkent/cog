package ast

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	"strconv"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Uint64Literal{}

type Uint64Literal struct {
	expression

	Token tokens.Token
	Value uint64
}

func NewUint64Literal(t tokens.Token) (*Uint64Literal, error) {
	value, err := strconv.ParseUint(t.Literal, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse int literal to int64: %w", err)
	}

	return &Uint64Literal{
		Token: t,
		Value: value,
	}, nil
}

func (l *Uint64Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Uint64Literal) Go() *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.INT,
		Value: strconv.FormatUint(l.Value, 10),
	}
}

func (l *Uint64Literal) Hash() uint64 {
	return hash(l)
}

func (l *Uint64Literal) String() string {
	return "(" + strconv.FormatUint(l.Value, 10) + " : uint64)"
}

func (l *Uint64Literal) Type() types.Type {
	return types.Basics[types.Uint64]
}
