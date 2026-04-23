package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Node = &Declaration{}

type Method struct {
	Token       tokens.Token
	Export      bool
	Receiver    *Identifier
	Type        types.Type
	Declaration *Declaration
}

func (s *Method) Kind() NodeKind {
	return KindMethod
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

	if s.Receiver != nil {
		_ = out.WriteByte('(')
		_, _ = out.WriteString(s.Receiver.Name)
		_, _ = out.WriteString(" : ")
		_, _ = out.WriteString(s.Type.String())
		_ = out.WriteByte(')')
	} else {
		_, _ = out.WriteString(s.Type.String())
	}

	_ = out.WriteByte('.')

	s.Declaration.Assignment.stringTo(out)
}

func (d *Method) String() string {
	var out strings.Builder
	d.stringTo(&out)

	return out.String()
}
