package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Switch{}

type Switch struct {
	Token      tokens.Token
	Label      *Label
	Identifier *Identifier // may be nil
	// Condition Expr // may be nil
	Cases   []*Case
	Default *Default // may be nil
}

func (s *Switch) Pos() (ln uint32, col uint16) {
	return s.Token.Ln, s.Token.Col
}

func (s *Switch) Hash() uint64 {
	return hash(s)
}

func (s *Switch) stringTo(out *strings.Builder) {
	if s.Label != nil {
		s.Label.stringTo(out)
		_ = out.WriteByte(' ')
	}

	_, _ = out.WriteString(s.Token.Type.String())
	_ = out.WriteByte(' ')

	if s.Identifier != nil {
		s.Identifier.stringTo(out)
	}

	_, _ = out.WriteString(" {\n")

	for _, c := range s.Cases {
		c.stringTo(out)
	}

	if s.Default != nil {
		s.Default.stringTo(out)
	}

	_ = out.WriteByte('}')
}

func (s *Switch) String() string {
	var out strings.Builder
	s.stringTo(&out)

	return out.String()
}

var _ Node = &Case{}

type Case struct {
	Token     tokens.Token
	Condition ExprValue
	Body      []NodeValue
}

func (c *Case) Pos() (ln uint32, col uint16) {
	return c.Token.Ln, c.Token.Col
}

func (c *Case) Hash() uint64 {
	return hash(c)
}

func (c *Case) stringTo(out *strings.Builder) {
	_, _ = out.WriteString(c.Token.Type.String())
	_ = out.WriteByte(' ')
	c.Condition.expr.stringTo(out)
	_, _ = out.WriteString(":\n")

	for _, stmt := range c.Body {
		_ = out.WriteByte('\t')
		stmt.node.stringTo(out)
		_ = out.WriteByte('\n')
	}
}

func (c *Case) String() string {
	var out strings.Builder
	c.stringTo(&out)

	return out.String()
}

var _ Node = &Default{}

type Default struct {
	Token tokens.Token
	Body  []NodeValue
}

func (d *Default) Pos() (ln uint32, col uint16) {
	return d.Token.Ln, d.Token.Col
}

func (d *Default) Hash() uint64 {
	return hash(d)
}

func (d *Default) stringTo(out *strings.Builder) {
	_, _ = out.WriteString(d.Token.Type.String())
	_, _ = out.WriteString(":\n")

	for _, stmt := range d.Body {
		_ = out.WriteByte('\t')
		stmt.node.stringTo(out)
		_ = out.WriteByte('\n')
	}
}

func (d *Default) String() string {
	var out strings.Builder
	d.stringTo(&out)

	return out.String()
}
