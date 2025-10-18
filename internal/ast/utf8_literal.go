package ast

import (
	goast "go/ast"
	gotoken "go/token"
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

func (l *UTF8Literal) Go() *goast.BasicLit {
	if strings.ContainsAny(l.Value, "\n\t") {
		return &goast.BasicLit{
			Kind:  gotoken.STRING,
			Value: "`" + l.Value + "`",
		}
	}

	return &goast.BasicLit{
		Kind:  gotoken.STRING,
		Value: `"` + l.Value + `"`,
	}
}

func (l *UTF8Literal) Hash() uint64 {
	return hash(l)
}

func (l *UTF8Literal) String() string {
	if strings.ContainsAny(l.Value, "\n\t") {
		return "(`" + l.Value + "` : utf8)"
	}

	return "(\"" + l.Value + "\" : utf8)"
}

func (l *UTF8Literal) Type() types.Type {
	return types.Basics[types.UTF8]
}
