package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Comment{}

type Comment struct {
	Token tokens.Token
	Text  string
}

func (c *Comment) Hash() uint64 {
	return hash(c)
}

func (c *Comment) Pos() (uint32, uint16) {
	return c.Token.Ln, c.Token.Col
}

func (c *Comment) stringTo(out *strings.Builder) {
	out.WriteString(c.Text)
}

func (c *Comment) String() string {
	var out strings.Builder
	c.stringTo(&out)

	return out.String()
}
