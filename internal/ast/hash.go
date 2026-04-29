package ast

import (
	"encoding/binary"
	"hash/maphash"
	"strings"
)

var seed = maphash.MakeSeed()

func hash(n Node) uint64 {
	ln, col := n.Pos()

	var out strings.Builder
	n.StringTo(&out, nil)
	str := out.String()

	b := make([]byte, 6, 6+len(str))
	binary.BigEndian.PutUint32(b[:4], ln)
	binary.BigEndian.PutUint16(b[4:6], col)
	b = append(b, str...)

	return maphash.Bytes(seed, b)
}
