package types

import (
	"strings"
)

type Union struct {
	Name     string // non-empty for builtin constraints (e.g., "int", "comparable")
	Variants []Type
	Exported bool
	Global   bool
}

func (t *Union) Kind() Kind {
	return UnionKind
}

func (t *Union) String() string {
	if t.Name != "" {
		return t.Name
	}

	var out strings.Builder

	for i, v := range t.Variants {
		if i > 0 {
			out.WriteString(" | ")
		}

		out.WriteString(v.String())
	}

	return out.String()
}

func (t *Union) Underlying() Type {
	return t
}
