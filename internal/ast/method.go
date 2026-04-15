package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Statement = &Declaration{}

type Method struct {
	statement

	Token       tokens.Token
	Receiver    *Identifier
	Reference   bool
	Declaration *Declaration
}

func (s *Method) Pos() (uint32, uint16) {
	return s.Token.Ln, s.Token.Col
}

func (s *Method) Hash() uint64 {
	return hash(s)
}

func (s *Method) stringTo(out *strings.Builder) {
	if s.Declaration.Assignment.Identifier.Exported {
		_, _ = out.WriteString("export ")
	}

	if s.Reference {
		_ = out.WriteByte('&')
	}

	_, _ = out.WriteString(s.Receiver.Name)
	_ = out.WriteByte('.')

	s.Declaration.Assignment.stringTo(out)
}

func (d *Method) String() string {
	var out strings.Builder
	d.stringTo(&out)

	return out.String()
}
