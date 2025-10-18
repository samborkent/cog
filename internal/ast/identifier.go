package ast

import (
	goast "go/ast"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Identifier{}

type Identifier struct {
	expression

	Token     tokens.Token
	Name      string
	ValueType types.Type
	Exported  bool
}

func (e *Identifier) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Identifier) Hash() uint64 {
	return hash(e)
}

func (e *Identifier) String() string {
	return e.Name
}

func (e *Identifier) Type() types.Type {
	if e.ValueType == nil {
		panic("identifier with nil type detected")
	}

	return e.ValueType
}

func (e *Identifier) Go() *goast.Ident {
	if e == nil || e.Name == "" {
		return nil
	}

	return &goast.Ident{
		Name: e.Name,
	}
}
