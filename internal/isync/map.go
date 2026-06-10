package isync

import (
	"iter"
	"maps"
	"sync"
)

type Map[K comparable, V any] struct {
	m          map[K]V
	mu         sync.RWMutex
	concurrent bool
}

// Option configures a Map at construction time.
type Option func(*bool)

// WithConcurrency enables RWMutex protection for concurrent access.
func WithConcurrency() Option {
	return func(concurrent *bool) {
		*concurrent = true
	}
}

func NewMap[K comparable, V any](opts ...Option) *Map[K, V] {
	m := &Map[K, V]{
		m: make(map[K]V),
	}

	for _, opt := range opts {
		opt(&m.concurrent)
	}

	return m
}

func (m *Map[K, V]) Map() map[K]V {
	return m.m
}

func (m *Map[K, V]) All() iter.Seq2[K, V] {
	if m.concurrent {
		return func(yield func(K, V) bool) {
			m.mu.RLock()
			defer m.mu.RUnlock()

			for k, v := range m.m {
				if !yield(k, v) {
					return
				}
			}
		}
	}

	return maps.All(m.m)
}

func (m *Map[K, V]) Load(key K) (V, bool) {
	if m.concurrent {
		m.mu.RLock()
		value, ok := m.m[key]
		m.mu.RUnlock()

		return value, ok
	}

	value, ok := m.m[key]

	return value, ok
}

func (m *Map[K, V]) Store(key K, value V) {
	if m.concurrent {
		m.mu.Lock()
		m.m[key] = value
		m.mu.Unlock()

		return
	}

	m.m[key] = value
}

func (m *Map[K, V]) Delete(key K) {
	if m.concurrent {
		m.mu.Lock()
		delete(m.m, key)
		m.mu.Unlock()

		return
	}

	delete(m.m, key)
}

func (m *Map[K, V]) Len() int {
	if m.concurrent {
		m.mu.RLock()
		length := len(m.m)
		m.mu.RUnlock()

		return length
	}

	return len(m.m)
}
