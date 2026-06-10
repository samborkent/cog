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

// Value gets an enum value by name. Returns nil if not found.
func (e *Enum) Value(name string) *EnumValue {
	for _, value := range e.Values {
		if value.Name == name {
			return value
		}
	}

	return nil
}

func (*Enum) Kind() Kind {
	return EnumKind
}

func (e *Enum) String() string {
	var str strings.Builder

	_, _ = str.WriteString("enum<")
	_, _ = str.WriteString(e.ValueType.String())
	_, _ = str.WriteString("> {")

	for i, val := range e.Values {
		if i == 0 {
			_, _ = str.WriteString("\n")
		}

		_, _ = str.WriteString(val.Name)
		_, _ = str.WriteString(" := ")
		_, _ = str.WriteString(val.Value.String)
		_, _ = str.WriteString(",\n")
	}

	_, _ = str.WriteString("}")

	return str.String()
}

func (e *Enum) Underlying() Type {
	return e
}
