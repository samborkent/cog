package types

import "strings"

var _ Type = &Enum{}

type Enum struct {
	ValueType Type
	Values    []*EnumValue
}

type EnumValue struct {
	Name  string
	Value Expression
}

func (*Enum) Kind() Kind {
	return EnumKind
}

func (e *Enum) String() string {
	// var out strings.Builder

	// _, _ = out.WriteString("({")

	// for i, val := range e.Values {
	// 	if i == 0 {
	// 		_ = out.WriteByte('\n')
	// 	}

	// 	_, _ = out.WriteString(val.Identifier.Name)
	// 	_, _ = out.WriteString(" := ")
	// 	_, _ = out.WriteString(val.Value.String())
	// 	_ = out.WriteByte('\n')
	// }

	// _, _ = out.WriteString("} : ")
	// _, _ = out.WriteString(e.Type().String())
	// _ = out.WriteByte(')')

	// return out.String()

	var str strings.Builder
	str.WriteString("enum<" + e.ValueType.String() + "> {")

	for i, val := range e.Values {
		if i == 0 {
			str.WriteString("\n")
		}

		str.WriteString(val.Name + " := " + val.Value.String() + ",\n")
	}

	str.WriteString("}")

	return str.String()
}

func (e *Enum) Underlying() Type {
	return e
}
