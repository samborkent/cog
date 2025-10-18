package types

import (
	"slices"
	"strings"
)

var _ Type = &Struct{}

type Struct struct {
	Fields []*Field
}

type Field struct {
	Name     string
	Type     Type
	Exported bool
}

func (s *Struct) Kind() Kind {
	return StructKind
}

func (s *Struct) Field(name string) *Field {
	index := slices.IndexFunc(s.Fields, func(field *Field) bool {
		return field.Name == name
	})
	if index == -1 {
		return nil
	}

	return s.Fields[index]
}

func (s *Struct) String() string {
	if len(s.Fields) == 0 {
		return "struct{}"
	}

	var out strings.Builder

	_, _ = out.WriteString("struct {\n")

	for _, field := range s.Fields {
		if field.Exported {
			_, _ = out.WriteString("export ")
		}

		_, _ = out.WriteString(field.Name)
		_, _ = out.WriteString(" : ")
		_, _ = out.WriteString(field.Type.String())
		_ = out.WriteByte('\n')
	}

	_ = out.WriteByte('}')

	return out.String()
}

func (s *Struct) Underlying() Type {
	return s
}
