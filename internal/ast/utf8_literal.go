package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type utf8 = string

var _ Expression = &UTF8Literal{}

type UTF8Literal struct {
	expression

	Token tokens.Token
	Value utf8
}

func NewUTF8Literal(t tokens.Token) *UTF8Literal {
	return &UTF8Literal{
		Token: t,
		Value: t.Literal,
	}
}

func (l *UTF8Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *UTF8Literal) Hash() uint64 {
	return hash(l)
}

func (l *UTF8Literal) stringTo(out *strings.Builder) {
	if strings.ContainsAny(l.Value, "\n\t") {
		_, _ = out.WriteString("(`")
		_, _ = out.WriteString(l.Value)
		_, _ = out.WriteString("` : utf8)")

		return
	}

	_, _ = out.WriteString("(\"")
	_, _ = out.WriteString(l.Value)
	_, _ = out.WriteString("\" : utf8)")
}

func (l *UTF8Literal) String() string {
	var out strings.Builder
	l.stringTo(&out)
	return out.String()
}

func (l *UTF8Literal) Type() types.Type {
	return types.Basics[types.UTF8]
}
