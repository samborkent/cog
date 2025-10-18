package tokens

import "fmt"

type Token struct {
	Type    Type
	Literal string
	Ln      uint32
	Col     uint16
}

func (t Token) String() string {
	if t.Literal == "" {
		return fmt.Sprintf("ln %d, col %d: %s",
			t.Ln, t.Col, t.Type,
		)
	}

	if t.Type == Builtin {
		return fmt.Sprintf("ln %d, col %d: @%s",
			t.Ln, t.Col, t.Literal,
		)
	}

	return fmt.Sprintf("ln %d, col %d: %s: %s",
		t.Ln, t.Col, t.Type, t.Literal,
	)
}
