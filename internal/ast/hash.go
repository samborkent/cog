package ast

import (
	"encoding/binary"
	"hash/maphash"
)

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
