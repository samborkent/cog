package tokens

import "fmt"

type Token struct {
	Type    Type
	Literal string
	FileID  uint16
	Ln      uint32
	Col     uint16
}

func (t Token) String() string {
	if t.Literal == "" {
		return fmt.Sprintf("file %d, ln %d, col %d: %s",
			t.FileID, t.Ln, t.Col, t.Type,
		)
	}

	if t.Type == Builtin {
		return fmt.Sprintf("file %d, ln %d, col %d: @%s",
			t.FileID, t.Ln, t.Col, t.Literal,
		)
	}

	return fmt.Sprintf("file %d, ln %d, col %d: %s: %s",
		t.FileID, t.Ln, t.Col, t.Type, t.Literal,
	)
}
