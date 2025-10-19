package ast

import (
	goast "go/ast"
	"strings"
	"unicode"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Identifier{}

type Identifier struct {
	expression

	Token     tokens.Token
	Name      string
	ValueType types.Type
	Exported  bool
}

func (e *Identifier) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Identifier) Hash() uint64 {
	return hash(e)
}

func (e *Identifier) String() string {
	return e.Name
}

func (e *Identifier) Type() types.Type {
	if e.ValueType == nil {
		panic("identifier with nil type detected")
	}

	return e.ValueType
}

func (e *Identifier) Go() *goast.Ident {
	if e == nil || e.Name == "" {
		return nil
	}

	return &goast.Ident{
		Name: convertExport(e.Name, e.Exported),
	}
}

func convertExport(ident string, exported bool) string {
	r := rune(ident[0])
	str := string(r)

	if exported {
		// If exported, ensure first letter is uppercase.
		str = strings.ToUpper(str)
	} else if unicode.IsUpper(r) {
		// If not exported, but first letter is uppercase, prefix it with underscore.
		str = "_" + str
	}

	if len(ident) > 1 {
		str += ident[1:]
	}

	return str
}
