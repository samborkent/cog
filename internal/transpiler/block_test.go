package transpiler_test

import "testing"

func TestConvertIfBlock(t *testing.T) {
	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	if true {
		@print("yes")
	}
}`)
		mustContain(t, got, "if true")
	})

	t.Run("if_break_generates_goto_label", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
ifLabel:
	if true {
		if true {
			break ifLabel
		}
	}
}`)
		mustContain(t, got, "goto")
	})
}

func TestConvertForBlock(t *testing.T) {
	t.Parallel()

	t.Run("block_scope", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	for {
		x := 1
		@print(x)
		break
	}
}`)
		mustContain(t, got, "for {")
	})
}
