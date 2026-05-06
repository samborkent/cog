package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type Qualifier uint8

const (
	QualifierType Qualifier = iota
	QualifierMethod
	QualifierImmutable
	QualifierVariable
	QualifierDynamic
	// QualifierConstant
)

var _ Expr = &Identifier{}

type Identifier struct {
	Token     tokens.Token
	Name      string
	ValueType types.Type
	Exported  bool
	Qualifier Qualifier
	Global    bool
}

func (e *Identifier) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Identifier) Hash() uint64 {
	return hash(e)
}

func (e *Identifier) StringTo(out *strings.Builder, _ *AST) {
	_, _ = out.WriteString(e.Name)
}

func (e *Identifier) String() string {
	return e.Name
}

func (e *Identifier) Type() types.Type {
	return e.ValueType
}
