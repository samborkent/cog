package ast

import (
	"encoding/binary"
	"hash/maphash"

	"github.com/samborkent/cog/internal/types"
)

type Node interface {
	Pos() (ln uint32, col uint16)
	Hash() uint64
	String() string
}

type Statement interface {
	Node
	statementNode()
}

type statement struct{}

func (statement) statementNode() {}

type Expression interface {
	Node
	Type() types.Type
	expressionNode()
}

type expression struct{}

func (expression) expressionNode() {}

var seed = maphash.MakeSeed()

func hash(n Node) uint64 {
	ln, col := n.Pos()
	str := n.String()

	b := make([]byte, 6, 6+len(str))
	binary.BigEndian.PutUint32(b[:4], ln)
	binary.BigEndian.PutUint16(b[4:6], col)
	b = append(b, str...)

	return maphash.Bytes(seed, b)
}
