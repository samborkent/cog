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
		_, _ = str.WriteString("error<")
		_, _ = str.WriteString(e.ValueType.String())
		_, _ = str.WriteString("> {")
	} else {
		_, _ = str.WriteString("error {")
	}

	for i, val := range e.Values {
		if i == 0 {
			_, _ = str.WriteString("\n")
		}

		if e.ValueType != nil {
			_, _ = str.WriteString(val.Name)
			_, _ = str.WriteString(" := ")
			_, _ = str.WriteString(val.Value.String)
			_, _ = str.WriteString(",\n")
		} else {
			_, _ = str.WriteString(val.Name)
			_, _ = str.WriteString(",\n")
		}
	}

	_, _ = str.WriteString("}")

	return str.String()
}

func (e *Error) Underlying() Type {
	return e
}
