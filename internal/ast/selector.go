package ast

import (
	"errors"
	"fmt"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Selector{}

type Selector struct {
	Token tokens.Token
	Expr  ExprValue // *Identifier or *Selector
	Field *Identifier
}

func (e *Selector) Kind() NodeKind {
	return KindSelector
}

func (e *Selector) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Selector) Hash() uint64 {
	return hash(e)
}

func (e *Selector) stringTo(out *strings.Builder) {
	e.Expr.expr.stringTo(out)
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
	current := e

selectorLoop:
	for {
		switch current.Expr.NodeKind {
		case KindSelector:
			current = current.Expr.expr.(*Selector)
			continue
		case KindIdentifier:
			leftMost = current.Expr.expr.(*Identifier)
			break selectorLoop
		default:
			return nil, fmt.Errorf("unexpected type %T found in selector expression", current.Expr)
		}
	}

	if leftMost == nil {
		return nil, errors.New("unable to find left-most identifier in selector expression")
	}

	return leftMost, nil
}
