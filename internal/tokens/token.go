package tokens

import (
	"strconv"
	"strings"
)

type Token struct {
	Literal string
	Ln      uint32
	Col     uint16
	Type    Type
}

func (t Token) String() string {
	var out strings.Builder
	t.StringTo(&out, "")
	return out.String()
}

func (t Token) StringTo(out *strings.Builder, fileName string) {
	if fileName != "" {
		_, _ = out.WriteString(fileName)
		_, _ = out.WriteString(": ")
	}

	_, _ = out.WriteString("ln ")
	_, _ = out.WriteString(strconv.FormatUint(uint64(t.Ln), 10))
	_, _ = out.WriteString(", col ")
	_, _ = out.WriteString(strconv.FormatUint(uint64(t.Col), 10))
	_, _ = out.WriteString(": ")

	if t.Type == Builtin {
		_ = out.WriteByte('@')
	}

	_, _ = out.WriteString(t.Type.String())

	if t.Literal != "" {
		_, _ = out.WriteString(": ")
		_, _ = out.WriteString(t.Literal)
	}
}
