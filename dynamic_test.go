package cog_test

import (
	"context"
	"testing"

	"github.com/petermattis/goid"
	"github.com/samborkent/cog"
)

var global string

func BenchmarkCtx(b *testing.B) {
	type valKey struct{}
	val := "default"
	ctx := context.WithValue(context.Background(), valKey{}, val)

	someFunc := func(ctx context.Context, _ string) {
		global = ctx.Value(valKey{}).(string)
		ctx = context.WithValue(ctx, valKey{}, "overwrite")
		global = ctx.Value(valKey{}).(string)
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

func BenchmarkDynamic(b *testing.B) {
	val := cog.NewDynamic("default")
	gid := goid.Get()
	val.Init(gid, 0)

	someFunc := func(pid int64, _ string) {
		gid := goid.Get()

		if pid != gid {
			val.Init(gid, pid)
			defer val.Delete(gid)
		}

		global = val.Get(gid)
		val.Put(gid, "overwrite")
		global = val.Get(gid)
	}

	for b.Loop() {
		done := make(chan struct{})

		go func() {
			someFunc(gid, "")

			close(done)
		}()

		<-done
	}
}
