package cog

import "hash/maphash"

var Seed = maphash.MakeSeed()

func HashASCII[Out ~uint64, In ~[]byte](in In) Out {
	return Out(maphash.Bytes(Seed, in))
}
