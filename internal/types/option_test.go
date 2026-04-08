package types

import "testing"

func TestOptionUnderlying(t *testing.T) {
	t.Parallel()

	opt := &Option{Value: Basics[Int64]}
	if opt.Underlying() != Basics[Int64] {
		t.Errorf("Option.Underlying() = %v, want int64", opt.Underlying())
	}
}
