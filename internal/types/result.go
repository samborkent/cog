package types

import (
	"strings"
)

type Result struct {
	Value, Error Type
}

func (t *Result) Kind() Kind {
	return ResultKind
}

func (t *Result) String() string {
	var out strings.Builder

	_, _ = out.WriteString(t.Value.String())
	_, _ = out.WriteString(" ! ")
	_, _ = out.WriteString(t.Error.String())

	return out.String()
}

func (t *Result) Underlying() Type {
	return t
}
