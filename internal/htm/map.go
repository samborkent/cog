package htm

import (
	"iter"
	"sync/atomic"

	"github.com/go4org/hashtriemap"
)

type Map[K comparable, V any] struct {
	m   *hashtriemap.HashTrieMap[K, V]
	len atomic.Int64
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		m: new(hashtriemap.HashTrieMap[K, V]),
	}
}

func (m *Map[K, V]) All() iter.Seq2[K, V] {
	return m.m.All()
}

func (m *Map[K, V]) Load(key K) (V, bool) {
	return m.m.Load(key)
}

func (m *Map[K, V]) Store(key K, value V) {
	m.m.Store(key, value)
	m.len.Add(1)
}

func (m *Map[K, V]) Delete(key K) {
	m.m.Delete(key)
	m.len.Add(-1)
}

func (m *Map[K, V]) Len() int {
	return int(m.len.Load())
}
