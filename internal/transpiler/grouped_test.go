package transpiler_test

import (
	"testing"
)

func TestGroupedExpression(t *testing.T) {
	t.Parallel()

	t.Run("simple_grouped", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	if (1 + 2) == 3 {}
}`)
		mustContain(t, got, "(1 + 2)")
	})

	t.Run("comparison_with_grouping", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := 5
	y := 10
	if (x == 5) != (y == 10) {}
}`)
		mustContain(t, got, "(x == 5)")
		mustContain(t, got, "(y == 10)")
	})

	t.Run("nested_grouped", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	if ((1 + 2) * 3) == 9 {}
}`)
		mustContain(t, got, "((1 + 2) * 3)")
	})

	t.Run("grouped_boolean", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	if (true && false) || (false && true) {}
}`)
		mustContain(t, got, "(true && false)")
		mustContain(t, got, "(false && true)")
	})
}
