package types

import (
	"testing"

	"github.com/samborkent/cog/internal/tokens"
)

func TestAnyType(t *testing.T) {
	t.Parallel()

	if Any.Kind() != AnyKind {
		t.Errorf("Any.Kind() = %v, want AnyKind", Any.Kind())
	}

	if Any.String() != "any" {
		t.Errorf("Any.String() = %q, want %q", Any.String(), "any")
	}

	if Any.Underlying() != Any {
		t.Error("Any.Underlying() != Any")
	}
}

func TestAnyNotInLookup(t *testing.T) {
	t.Parallel()

	// any is a constraint, not a standalone type — it must not be in Lookup.
	if _, ok := Lookup[tokens.Any]; ok {
		t.Error("tokens.Any should not be in Lookup map (any is constraint-only)")
	}
}

func TestEqualAny(t *testing.T) {
	t.Parallel()

	if !Equal(Any, Any) {
		t.Error("Equal(any, any) = false")
	}

	if Equal(Any, Basics[Int64]) {
		t.Error("Equal(any, int64) = true")
	}

	if Equal(Basics[Int64], Any) {
		t.Error("Equal(int64, any) = true")
	}
}
