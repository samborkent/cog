package transpiler

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/samborkent/cog/internal/lexer"
	"github.com/samborkent/cog/internal/parser"
)

// TestTranspileLineDirectives is an end-to-end test that runs the lexer,
// parser and transpiler on the example.cog file and asserts that the
// generated Go source contains //line directives pointing back to the
// original .cog source for statement locations.
func TestTranspileLineDirectives(t *testing.T) {
	f, err := os.Open("../../example.cog")
	if err != nil {
		t.Fatalf("open example.cog: %v", err)
	}
	defer f.Close()

	l := lexer.NewLexer(f)
	toks, err := l.Parse(context.Background())
	if err != nil {
		t.Fatalf("lexer parse: %v", err)
	}

	p, err := parser.NewParser(toks, false)
	if err != nil {
		t.Fatalf("new parser: %v", err)
	}

	astf, err := p.Parse(context.Background())
	if err != nil {
		t.Fatalf("parser parse: %v", err)
	}

	tp := NewTranspiler(astf, "example.cog")
	src, err := tp.Transpile()
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}

	// We expect at least one //line directive referencing example.cog
	if !strings.Contains(src, "//line example.cog:") {
		t.Fatalf("no //line directives found in transpiled source")
	}

	// Ensure a specific statement (print of c1) has a preceding //line.
	re := regexp.MustCompile(`//line example.cog:\d+\s*cog\.Print\(c1\)`)
	if !re.MatchString(src) {
		// If this fails, write src to test output for debugging.
		t.Fatalf("expected a //line before cog.Print(c1); transpiled source snippet:\n%s", src)
	}
}
