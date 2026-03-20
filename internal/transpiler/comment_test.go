package transpiler_test

import "testing"

func TestCommentTranspile(t *testing.T) {
	t.Parallel()

	t.Run("top_level_line_comment", func(t *testing.T) {
		t.Parallel()
		got := transpileWithPrint(t, `package p
// top level comment
main : proc() = {}`)
		mustContain(t, got, "// top level comment")
		mustNotContain(t, got, "__cogReplaceMe")
	})

	t.Run("top_level_block_comment", func(t *testing.T) {
		t.Parallel()
		got := transpileWithPrint(t, `package p
/* block comment */
main : proc() = {}`)
		mustContain(t, got, "/* block comment */")
		mustNotContain(t, got, "__cogReplaceMe")
		mustNotContain(t, got, "__cog_block__")
	})

	t.Run("inline_comment_becomes_standalone", func(t *testing.T) {
		t.Parallel()
		// Inline comments get tokenized as separate statements,
		// so they appear on their own line after the declaration.
		got := transpileWithPrint(t, `package p
main : proc() = {
	x := 5 // inline note
	@print(x)
}`)
		mustContain(t, got, "// inline note")
		mustNotContain(t, got, "__cogReplaceMe")
	})

	t.Run("multi_line_block_comment", func(t *testing.T) {
		t.Parallel()
		got := transpileWithPrint(t, `package p
/* multi
   line
   comment */
main : proc() = {}`)
		mustContain(t, got, "/* multi")
		mustContain(t, got, "   comment */")
		mustNotContain(t, got, "__cogReplaceMe")
		mustNotContain(t, got, "__cog_block__")
	})

	t.Run("comment_inside_block", func(t *testing.T) {
		t.Parallel()
		got := transpileWithPrint(t, `package p
main : proc() = {
	// inside block
	x := 1
	@print(x)
}`)
		mustContain(t, got, "// inside block")
		mustNotContain(t, got, "__cogReplaceMe")
	})

	t.Run("multiple_comments", func(t *testing.T) {
		t.Parallel()
		got := transpileWithPrint(t, `package p
// first comment
// second comment
main : proc() = {}`)
		mustContain(t, got, "// first comment")
		mustContain(t, got, "// second comment")
		mustNotContain(t, got, "__cogReplaceMe")
	})

	t.Run("dyn_inline_comment", func(t *testing.T) {
		t.Parallel()
		got := transpileWithPrint(t, `package p
dyn val : utf8 = "default" // dyn comment
main : proc() = {}`)
		mustContain(t, got, "// dyn comment")
		mustNotContain(t, got, "__cogReplaceMe")
	})
}
