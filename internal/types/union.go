package types

import (
	"strings"
)

const UnionNumTypes = 2

type Union struct {
	Either, Or Type
	Exported   bool
}

func (t *Union) Kind() Kind {
	return UnionKind
}

func (t *Union) String() string {
	var out strings.Builder

	_, _ = out.WriteString(t.Either.String())
	_, _ = out.WriteString(" | ")
	_, _ = out.WriteString(t.Or.String())

	return out.String()
}

func (t *Union) Underlying() Type {
	return t
}
