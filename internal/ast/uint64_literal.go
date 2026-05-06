package ast

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Uint64Literal{}

type Uint64Literal struct {
	Token tokens.Token
	Value uint64
}

func (a *AST) NewUint64Literal(t tokens.Token) (ExprIndex, error) {
	value, err := strconv.ParseUint(t.Literal, 10, 64)
	if err != nil {
		return ZeroExprIndex, fmt.Errorf("unable to parse int literal to uint64: %w", err)
	}

	expr := New[Uint64Literal](a)
	expr.Token = t
	expr.Value = value

	return a.AddExpr(expr), nil
}

func (l *Uint64Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Uint64Literal) Hash() uint64 {
	return hash(l)
}

func (l *Uint64Literal) StringTo(out *strings.Builder, _ *AST) {
	_ = out.WriteByte('(')
	_, _ = out.WriteString(strconv.FormatUint(l.Value, 10))
	_, _ = out.WriteString(" : uint64)")
}

func (l *Uint64Literal) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *Uint64Literal) Type() types.Type {
	return types.Basics[types.Uint64]
}
