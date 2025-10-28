package cog_test

import (
	"context"
	"hash/maphash"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/petermattis/goid"
)

var global string

func BenchmarkGOID(b *testing.B) {
	val := make(map[int64]string)
	var valLock sync.Mutex

	deleteVal := func(gid int64) {
		valLock.Lock()
		delete(val, gid)
		valLock.Unlock()
	}

	gid := goid.Get()
	val[gid] = "default"
	defer deleteVal(gid)

	for b.Loop() {
		done := make(chan struct{})

		go func(pid int64) {
			gid := goid.Get()
			valLock.Lock()
			val[gid] = val[pid]
			valLock.Unlock()
			defer deleteVal(gid)

			someFunc := func(_ string) {
				global = val[gid]
				val[gid] = "overwrite"
				global = val[gid]
			}

			someFunc("")

			close(done)
		}(gid)

		<-done
	}
}

func BenchmarkGOIDParent(b *testing.B) {
	val := make(map[int64]string)
	var valLock sync.Mutex

	deleteVal := func(gid int64) {
		valLock.Lock()
		delete(val, gid)
		valLock.Unlock()
	}

	someFunc := func(pid int64, _ string) {
		gid := goid.Get()

		_, ok := val[gid]
		if ok {
			val[gid] = val[pid]
		} else {
			valLock.Lock()
			val[gid] = val[pid]
			valLock.Unlock()
		}

		defer deleteVal(gid)

		global = val[gid]
		val[gid] = "overwrite"
		global = val[gid]
	}

	gid := goid.Get()
	val[gid] = "default"
	defer deleteVal(gid)

	for b.Loop() {
		done := make(chan struct{})

		go func() {
			someFunc(gid, "")

			close(done)
		}()

		<-done
	}
}

func BenchmarkCtx(b *testing.B) {
	type valKey struct{}
	val := "default"
	ctx := context.WithValue(context.Background(), valKey{}, val)

	someFunc := func(ctx context.Context, _ string) {
		val := ctx.Value(valKey{}).(string)
		global = val

		val = "overwrite"
		ctx = context.WithValue(ctx, valKey{}, val)
		global = val
	}

	for b.Loop() {
		done := make(chan struct{})

		go func() {
			someFunc(ctx, "")

			close(done)
		}()

		<-done
	}
}

var globalGOID int64

func BenchmarkGetGOID(b *testing.B) {
	gid := goid.Get()
	globalGOID = gid

	for b.Loop() {
		gid := goid.Get()
		globalGOID = gid
	}
}

func TestGetGOID(t *testing.T) {
	gid := goid.Get()
	t.Logf("gid 1: %d", gid)

	synctest.Test(t, func(t *testing.T) {
		go func() {
			gid := goid.Get()
			t.Logf("gid 2: %d", gid)

			go func() {
				gid := goid.Get()
				t.Logf("gid 3: %d", gid)
			}()
		}()

		go func() {
			gid := goid.Get()
			t.Logf("gid 4: %d", gid)
		}()

		go func() {
			gid := goid.Get()
			t.Logf("gid 4: %d", gid)
		}()

		synctest.Wait()
	})
}

var seed = maphash.MakeSeed()

func BenchmarkCogContext(b *testing.B) {
	type Context struct {
		values map[uint64]any
	}

	ctx := &Context{
		values: make(map[uint64]any),
	}
	valHash := maphash.String(seed, "val")
	ctx.values[valHash] = "default"

	someFunc := func(ctx *Context, _ string) {
		val := ctx.values[valHash].(string)
		global = val

		ctx.values[valHash] = "overwrite"
		val = ctx.values[valHash].(string)
		global = val
	}

	for b.Loop() {
		done := make(chan struct{})

		go func() {
			someFunc(ctx, "")

			close(done)
		}()

		<-done
	}
}
