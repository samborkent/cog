package types

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

	// TODO: use strings.Builder (copy ast.EnumLiteral logic, then delete)
	str := "enum[" + e.ValueType.String() + "] {"

	for i, val := range e.Values {
		if i == 0 {
			str += "\n"
		}

		str += val.Name + " := " + val.Value.String() + ",\n"
	}

	str += "}"

	return str
}

func (e *Enum) Underlying() Type {
	return e
}
