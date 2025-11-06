package types

import (
	"strings"
)

type Procedure struct {
	Function   bool
	Parameters []*Parameter
	ReturnType Type // may be nil
}

type Parameter struct {
	Name     string
	Optional bool
	Type     Type
	Default  Expression // cannot be ast.Expression due to import cycle
}

func (p *Procedure) Kind() Kind {
	return ProcedureKind
}

func (p *Procedure) String() string {
	var out strings.Builder

	if p.Function {
		_, _ = out.WriteString("func(")
	} else {
		_, _ = out.WriteString("proc(")
	}

	for i, param := range p.Parameters {
		_, _ = out.WriteString(param.Name)

		if param.Optional {
			_ = out.WriteByte('?')
		}

		_, _ = out.WriteString(" : ")
		_, _ = out.WriteString(param.Type.String())

		if param.Default != nil {
			_, _ = out.WriteString(" = ")
			_, _ = out.WriteString(param.Default.String())
		}

		if i < len(p.Parameters)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_ = out.WriteByte(')')

	if p.ReturnType != nil {
		_ = out.WriteByte(' ')
		_, _ = out.WriteString(p.ReturnType.String())
	}

	return out.String()
}

func (p *Procedure) Underlying() Type {
	return p
}
