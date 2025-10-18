package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Statement = &Block{}

type Block struct {
	statement

	Start, End tokens.Token
	Statements []Statement
}

func (b *Block) Pos() (uint32, uint16) {
	return b.Start.Ln, b.Start.Col
}

func (b *Block) Hash() uint64 {
	return hash(b)
}

func (b *Block) String() string {
	var out strings.Builder

	_ = out.WriteByte('{')

	for i, stmt := range b.Statements {
		if i == 0 {
			_ = out.WriteByte('\n')
		}

		_ = out.WriteByte('\t')
		_, _ = out.WriteString(stmt.String())
		_ = out.WriteByte('\n')
	}

	_ = out.WriteByte('}')

	return out.String()
}
