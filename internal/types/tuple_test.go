package types

import "testing"

func TestTupleIndexBounds(t *testing.T) {
	t.Parallel()

	tup := &Tuple{Types: []Type{Basics[Int64], Basics[UTF8]}}

	if tup.Index(0).Kind() != Int64 {
		t.Errorf("Index(0) = %s, want int64", tup.Index(0))
	}

	if tup.Index(1).Kind() != UTF8 {
		t.Errorf("Index(1) = %s, want utf8", tup.Index(1))
	}

	// Negative index should panic.
	t.Run("negative_panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for negative index")
			}
		}()

		tup.Index(-1)
	})

	// Index equal to length should panic.
	t.Run("at_length_panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for index == len")
			}
		}()

		tup.Index(2)
	})
}
