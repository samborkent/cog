package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Block{}

type Block struct {
	Start, End tokens.Token
	Statements []NodeIndex
}

func (a *AST) NewBlock(start, end tokens.Token, stmts []NodeIndex) NodeIndex {
	block := New[Block](a)
	block.Start = start
	block.End = end
	block.Statements = stmts
	return a.AddNode(block)
}

func (n *Block) Pos() (uint32, uint16) {
	return n.Start.Ln, n.Start.Col
}

func (n *Block) Hash() uint64 {
	return hash(n)
}

func (n *Block) StringTo(out *strings.Builder, a *AST) {
	_ = out.WriteByte('{')

	for i, stmt := range n.Statements {
		if i == 0 {
			_ = out.WriteByte('\n')
		}

		_ = out.WriteByte('\t')
		a.nodes[stmt].StringTo(out, a)
		_ = out.WriteByte('\n')
	}

	_ = out.WriteByte('}')
}
