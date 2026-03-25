package cog

import (
	"os"
	"os/signal"
	"slices"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Signal struct {
	mu     sync.Mutex
	child  []*Signal
	sig    chan struct{}
	closed atomic.Bool
}

func NewSignal(parent *Signal) *Signal {
	sig := &Signal{
		sig: make(chan struct{}),
	}

	parent.mu.Lock()
	parent.child = append(parent.child, sig)
	parent.mu.Unlock()

	return sig
}

func NewMainSignal() *Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	sig := &Signal{
		sig: make(chan struct{}),
	}

	go func() {
		defer close(c)

		<-c
		sig.Cancel()
	}()

	return sig
}

func (s *Signal) Cancel() {
	// close in backwards order
	for i := range slices.Backward(s.child) {
		if !s.child[i].closed.Load() {
			s.child[i].closed.Store(true)
			close(s.child[i].sig)
		}
	}

	if !s.closed.Load() {
		s.closed.Store(true)
		close(s.sig)
	}
}

func (s *Signal) Done() <-chan struct{} {
	return s.sig
}

func (s *Signal) Timeout(d time.Duration) *Signal {
	sig := NewSignal(s)

	go func() {
		<-time.After(d)
		sig.Cancel()
	}()

	return sig
}
