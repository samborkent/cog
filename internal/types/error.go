package types

import "strings"

var _ Type = &Error{}

// Error represents an error enum type. Typeless errors (ValueType == nil)
// print as their variant name. Typed errors require ascii or utf8 value type.
// TODO: In the future, also allow interface{ String() string } and interface{ Error() string }.
type Error struct {
	ValueType Type // nil for typeless, ASCII or UTF8 for typed
	Values    []*EnumValue
}

func (*Error) Kind() Kind {
	return ErrorKind
}

func (e *Error) String() string {
	var str strings.Builder

	if e.ValueType != nil {
		str.WriteString("error<" + e.ValueType.String() + "> {")
	} else {
		str.WriteString("error {")
	}

	for i, val := range e.Values {
		if i == 0 {
			str.WriteString("\n")
		}

		if e.ValueType != nil {
			str.WriteString(val.Name + " := " + val.Value.String + ",\n")
		} else {
			str.WriteString(val.Name + ",\n")
		}
	}

	str.WriteString("}")

	return str.String()
}

func (e *Error) Underlying() Type {
	return e
}
