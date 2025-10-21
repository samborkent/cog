package types

import (
	"fmt"
	"strings"
)

const (
	TupleMinTypes = 2
	TupleMaxTypes = 8
)

type Tuple struct {
	Types    []Type
	Exported bool
}

func (t *Tuple) Index(i int) Type {
	if i > TupleMaxTypes || i > len(t.Types) {
		panic("tuple type index out-of-range")
	}

	return t.Types[i]
}

func (t *Tuple) Kind() Kind {
	return TupleKind
}

func (t *Tuple) String() string {
	if len(t.Types) < TupleMinTypes || len(t.Types) > TupleMaxTypes {
		panic(fmt.Sprintf("Tuple.String: tuple must have 2 to 8 types, got %d", len(t.Types)))
	}

	var out strings.Builder

	for i, typ := range t.Types {
		_, _ = out.WriteString(typ.String())

		if i < len(t.Types)-1 {
			_, _ = out.WriteString(" & ")
		}
	}

	return out.String()
}

func (t *Tuple) Underlying() Type {
	return t
}
