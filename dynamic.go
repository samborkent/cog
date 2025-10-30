package cog

import "sync"

// Dynamic implements a dynamically scoped variable.
// Values are addressed by goid's to achieve thread local storage.
type Dynamic[T any] struct {
	val map[int64]T
	mu  sync.Mutex
}

func NewDynamic[T any](initial T) *Dynamic[T] {
	val := make(map[int64]T)
	val[0] = initial

	return &Dynamic[T]{
		val: val,
	}
}

func (d *Dynamic[T]) Get(gid int64) T {
	return d.val[gid]
}

func (d *Dynamic[T]) Init(gid, pid int64) {
	d.Put(gid, d.Get(pid))
}

func (d *Dynamic[T]) Put(gid int64, val T) {
	_, ok := d.val[gid]
	if ok {
		d.val[gid] = val
	} else {
		d.mu.Lock()
		d.val[gid] = val
		d.mu.Unlock()
	}
}

func (d *Dynamic[T]) Delete(gid int64) {
	d.mu.Lock()
	delete(d.val, gid)
	d.mu.Unlock()
}
