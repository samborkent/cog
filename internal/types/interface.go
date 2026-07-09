package types

import "strings"

var EmptyInterface = &Interface{}

type Interface struct {
	TypeParams []*Alias
	Methods    []*Method
}

type Method struct {
	Name      string
	Procedure *Procedure
}

// Method finds method by name, returns nil if method doesn't exist.
func (i *Interface) Method(name string) *Method {
	for _, method := range i.Methods {
		if method.Name == name {
			return method
		}
	}

	return nil
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
