package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Selector{}

type Selector struct {
	expression

	Token      tokens.Token
	Identifier *Identifier
	Field      *Identifier
}

func (e *Selector) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Selector) Hash() uint64 {
	return hash(e)
}

func (e *Selector) stringTo(out *strings.Builder) {
	_, _ = out.WriteString(e.Identifier.Name)
	_ = out.WriteByte('.')
	_, _ = out.WriteString(e.Field.Name)
}

func (e *Selector) String() string {
	var out strings.Builder
	e.stringTo(&out)

	return out.String()
}

func (e *Selector) Type() types.Type {
	return e.Field.Type()
}
