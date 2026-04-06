package ast

import (
	"errors"
	"fmt"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Selector{}

type Selector struct {
	expression

	Token      tokens.Token
	Expression Expression // *Identifier or *Selector
	Field      *Identifier
}

func (e *Selector) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Selector) Hash() uint64 {
	return hash(e)
}

func (e *Selector) stringTo(out *strings.Builder) {
	e.Expression.stringTo(out)
	_ = out.WriteByte('.')
	e.Field.stringTo(out)
}

func (e *Selector) String() string {
	var out strings.Builder
	e.stringTo(&out)

	return out.String()
}

func (e *Selector) Type() types.Type {
	return e.Field.Type()
}

func (e *Selector) LeftMost() (*Identifier, error) {
	var leftMost *Identifier

	// Find left-most identifier of selector.
selectorLoop:
	for {
		switch sel := e.Expression.(type) {
		case *Selector:
			continue
		case *Identifier:
			leftMost = sel
			break selectorLoop
		default:
			return nil, fmt.Errorf("unexpected type %T found in selector expression", e.Expression)
		}
	}

	if leftMost == nil {
		return nil, errors.New("unable to find left-most identifier in selector expression")
	}

	return leftMost, nil
}
