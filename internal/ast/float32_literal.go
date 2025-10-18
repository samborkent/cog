package ast

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	"strconv"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Float32Literal{}

type Float32Literal struct {
	expression

	Token tokens.Token
	Value float32
}

func NewFloat32Literal(t tokens.Token) (*Float32Literal, error) {
	value, err := strconv.ParseFloat(t.Literal, 32)
	if err != nil {
		return nil, fmt.Errorf("unable to parse float literal to float32: %w", err)
	}

	return &Float32Literal{
		Token: t,
		Value: float32(value),
	}, nil
}

func (l *Float32Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Float32Literal) Go() *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.FLOAT,
		Value: strconv.FormatFloat(float64(l.Value), 'g', -1, 32),
	}
}

func (l *Float32Literal) Hash() uint64 {
	return hash(l)
}

func (l *Float32Literal) String() string {
	return "(" + strconv.FormatFloat(float64(l.Value), 'g', -1, 32) + " : float32)"
}

func (l *Float32Literal) Type() types.Type {
	return types.Basics[types.Float32]
}
