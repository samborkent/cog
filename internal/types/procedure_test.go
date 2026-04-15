package types

import "testing"

func TestProcedureStringWithTypeParams(t *testing.T) {
	t.Parallel()

	t.Run("single_constraint", func(t *testing.T) {
		t.Parallel()

		p := &Procedure{
			Function: true,
			TypeParams: []*Alias{
				{Name: "T", Constraint: Any},
			},
			Parameters: []*Parameter{
				{Name: "x", Type: Basics[Int64]},
			},
			ReturnType: Basics[Int64],
		}
		got := p.String()

		want := "func<T ~ any>(x : int64) int64"
		if got != want {
			t.Errorf("Procedure.String() = %q, want %q", got, want)
		}
	})

	t.Run("multi_constraint", func(t *testing.T) {
		t.Parallel()

		p := &Procedure{
			Function: true,
			TypeParams: []*Alias{
				{Name: "T", Constraint: &Union{Variants: []Type{Constraints["string"], Constraints["int"]}}},
			},
			Parameters: []*Parameter{},
		}
		got := p.String()

		want := "func<T ~ string | int>()"
		if got != want {
			t.Errorf("Procedure.String() = %q, want %q", got, want)
		}
	})

	t.Run("multiple_type_params", func(t *testing.T) {
		t.Parallel()

		p := &Procedure{
			TypeParams: []*Alias{
				{Name: "K", Constraint: Constraints["comparable"]},
				{Name: "V", Constraint: Any},
			},
			Parameters: []*Parameter{},
		}
		got := p.String()

		want := "proc<K ~ comparable, V ~ any>()"
		if got != want {
			t.Errorf("Procedure.String() = %q, want %q", got, want)
		}
	})
}
