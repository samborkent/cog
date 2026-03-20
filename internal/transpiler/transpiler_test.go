package transpiler_test

import "testing"

func TestTranspile(t *testing.T) {
	t.Parallel()

	t.Run("package_name", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, "package myapp\nmain : proc() = {}")
		mustContain(t, got, "package myapp")
	})

	t.Run("go_import", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, "package p\ngoimport (\n\t\"strings\"\n)\nmain : proc() = {\n\tx := @go.strings.ToUpper(\"hello\")\n\t@print(x)\n}")
		mustContain(t, got, "\"strings\"")
	})

	t.Run("line_directive", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, "package p\nx := 42\nmain : proc() = {}")
		mustContain(t, got, "//line test.cog:")
	})

	t.Run("builtin_import_added", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, "package p\nmain : proc() = {\n\t@print(\"hello\")\n}")
		mustContain(t, got, "\"github.com/samborkent/cog/builtin\"")
	})
}
