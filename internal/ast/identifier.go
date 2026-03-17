package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type Qualifier uint8

const (
	QualifierType Qualifier = iota
	QualifierImmutable
	QualifierVariable
	QualifierDynamic
	// QualifierConstant
)

var _ Expression = &Identifier{}

type Identifier struct {
	expression

	Token     tokens.Token
	Name      string
	ValueType types.Type
	Exported  bool
	Qualifier Qualifier
}

func (e *Identifier) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Identifier) Hash() uint64 {
	return hash(e)
}

func (e *Identifier) stringTo(out *strings.Builder) {
	_, _ = out.WriteString(e.Name)
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
