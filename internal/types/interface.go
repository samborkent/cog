package types

import "strings"

type Interface struct {
	Methods []*Method
}

type Method struct {
	Name      string
	Procedure *Procedure
}

func (n *Interface) Kind() Kind {
	return InterfaceKind
}

func (n *Interface) String() string {
	var out strings.Builder

	_, _ = out.WriteString("interface {")

	for i, method := range n.Methods {
		_, _ = out.WriteString(method.Name)
		_, _ = out.WriteString(" : ")
		_, _ = out.WriteString(method.Procedure.String())

		if i < len(n.Methods)-1 {
			_ = out.WriteByte('\n')
		}
	}

	_ = out.WriteByte('}')

	return out.String()
}

func (n *Interface) Underlying() Type {
	return n
}
