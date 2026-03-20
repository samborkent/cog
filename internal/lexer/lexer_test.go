package lexer

import (
	"context"
	"strings"
	"testing"

	"github.com/samborkent/cog/internal/tokens"
)

func lex(t *testing.T, src string) []tokens.Token {
	t.Helper()
	l := NewLexer(strings.NewReader(src))
	toks, err := l.Parse(t.Context())
	if err != nil {
		t.Fatalf("unexpected lex error: %v", err)
	}
	return toks
}

func lexOne(t *testing.T, src string) tokens.Token {
	t.Helper()
	toks := lex(t, src)
	if len(toks) < 2 {
		t.Fatalf("expected at least 1 token + EOF, got %d tokens", len(toks))
	}
	if toks[len(toks)-1].Type != tokens.EOF {
		t.Fatalf("expected last token to be EOF, got %s", toks[len(toks)-1].Type)
	}
	return toks[0]
}

func TestSingleCharTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		src      string
		expected tokens.Type
	}{
		{"(", tokens.LParen},
		{")", tokens.RParen},
		{"{", tokens.LBrace},
		{"}", tokens.RBrace},
		{"[", tokens.LBracket},
		{"]", tokens.RBracket},
		{",", tokens.Comma},
		{".", tokens.Dot},
		{":", tokens.Colon},
		{"+", tokens.Plus},
		{"-", tokens.Minus},
		{"*", tokens.Asterisk},
		{"/", tokens.Divide},
		{"?", tokens.Question},
		{"~", tokens.Tilde},
		{"^", tokens.BitXor},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			t.Parallel()
			tok := lexOne(t, tt.src)
			if tok.Type != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tok.Type)
			}
		})
	}
}

func TestMultiCharTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      string
		expected tokens.Type
	}{
		{"equal", "==", tokens.Equal},
		{"not_equal", "!=", tokens.NotEqual},
		{"gt_equal", ">=", tokens.GTEqual},
		{"lt_equal", "<=", tokens.LTEqual},
		{"declaration", ":=", tokens.Declaration},
		{"and", "&&", tokens.And},
		{"or", "||", tokens.Or},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tok := lexOne(t, tt.src)
			if tok.Type != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tok.Type)
			}
		})
	}
}

func TestSingleVsMultiCharDisambiguation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      string
		expected tokens.Type
	}{
		{"assign_alone", "= x", tokens.Assign},
		{"not_alone", "! x", tokens.Not},
		{"gt_alone", "> x", tokens.GT},
		{"lt_alone", "< x", tokens.LT},
		{"colon_alone", ": x", tokens.Colon},
		{"bitand_alone", "& x", tokens.BitAnd},
		{"pipe_alone", "| x", tokens.Pipe},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tok := lexOne(t, tt.src)
			if tok.Type != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tok.Type)
			}
		})
	}
}

func TestIntLiteral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		src     string
		literal string
	}{
		{"0", "0"},
		{"42", "42"},
		{"1000000", "1000000"},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			t.Parallel()
			tok := lexOne(t, tt.src)
			if tok.Type != tokens.IntLiteral {
				t.Errorf("expected IntLiteral, got %s", tok.Type)
			}
			if tok.Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tok.Literal)
			}
		})
	}
}

func TestFloatLiteral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		src     string
		literal string
	}{
		{"3.14", "3.14"},
		{"0.5", "0.5"},
		{"1.0e10", "1.0e10"},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			t.Parallel()
			tok := lexOne(t, tt.src)
			if tok.Type != tokens.FloatLiteral {
				t.Errorf("expected FloatLiteral, got %s", tok.Type)
			}
			if tok.Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tok.Literal)
			}
		})
	}
}

func TestStringLiteral(t *testing.T) {
	t.Parallel()

	tok := lexOne(t, `"hello world"`)
	if tok.Type != tokens.StringLiteral {
		t.Errorf("expected StringLiteral, got %s", tok.Type)
	}
	if tok.Literal != "hello world" {
		t.Errorf("expected literal %q, got %q", "hello world", tok.Literal)
	}
}

func TestRawStringLiteral(t *testing.T) {
	t.Parallel()

	tok := lexOne(t, "`raw string`")
	if tok.Type != tokens.StringLiteral {
		t.Errorf("expected StringLiteral, got %s", tok.Type)
	}
	if tok.Literal != "raw string" {
		t.Errorf("expected literal %q, got %q", "raw string", tok.Literal)
	}
}

func TestEmptyStringLiteral(t *testing.T) {
	t.Parallel()

	tok := lexOne(t, `""`)
	if tok.Type != tokens.StringLiteral {
		t.Errorf("expected StringLiteral, got %s", tok.Type)
	}
	if tok.Literal != "" {
		t.Errorf("expected empty literal, got %q", tok.Literal)
	}
}

func TestKeywords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		src      string
		expected tokens.Type
	}{
		{"package", tokens.Package},
		{"import", tokens.Import},
		{"export", tokens.Export},
		{"goimport", tokens.GoImport},
		{"proc", tokens.Procedure},
		{"func", tokens.Function},
		{"var", tokens.Variable},
		{"dyn", tokens.Dynamic},
		{"if", tokens.If},
		{"else", tokens.Else},
		{"for", tokens.For},
		{"switch", tokens.Switch},
		{"case", tokens.Case},
		{"default", tokens.Default},
		{"return", tokens.Return},
		{"break", tokens.Break},
		{"continue", tokens.Continue},
		{"in", tokens.In},
		{"async", tokens.Async},
		{"true", tokens.True},
		{"false", tokens.False},
		{"struct", tokens.Struct},
		{"enum", tokens.Enum},
		{"map", tokens.Map},
		{"set", tokens.Set},
		{"error", tokens.Error},
		{"any", tokens.Any},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			t.Parallel()
			tok := lexOne(t, tt.src)
			if tok.Type != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tok.Type)
			}
		})
	}
}

func TestTypeKeywords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		src      string
		expected tokens.Type
	}{
		{"ascii", tokens.ASCII},
		{"utf8", tokens.UTF8},
		{"bool", tokens.Bool},
		{"uint8", tokens.Uint8},
		{"uint16", tokens.Uint16},
		{"uint32", tokens.Uint32},
		{"uint64", tokens.Uint64},
		{"uint128", tokens.Uint128},
		{"int8", tokens.Int8},
		{"int16", tokens.Int16},
		{"int32", tokens.Int32},
		{"int64", tokens.Int64},
		{"int128", tokens.Int128},
		{"float16", tokens.Float16},
		{"float32", tokens.Float32},
		{"float64", tokens.Float64},
		{"complex32", tokens.Complex32},
		{"complex64", tokens.Complex64},
		{"complex128", tokens.Complex128},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			t.Parallel()
			tok := lexOne(t, tt.src)
			if tok.Type != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tok.Type)
			}
		})
	}
}

func TestIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		src     string
		literal string
	}{
		{"foo", "foo"},
		{"myVar", "myVar"},
		{"x123", "x123"},
		{"_underscore", "_underscore"},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			t.Parallel()
			tok := lexOne(t, tt.src)
			if tok.Type != tokens.Identifier {
				t.Errorf("expected Identifier, got %s", tok.Type)
			}
			if tok.Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tok.Literal)
			}
		})
	}
}

func TestBuiltinToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		src     string
		literal string
	}{
		{"@print", "print"},
		{"@if", "if"},
		{"@alloc", "alloc"},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			t.Parallel()
			tok := lexOne(t, tt.src)
			if tok.Type != tokens.Builtin {
				t.Errorf("expected Builtin, got %s", tok.Type)
			}
			if tok.Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tok.Literal)
			}
		})
	}
}

func TestCommentsAreSkipped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
	}{
		{"line_comment", "// this is a comment\n42"},
		{"block_comment", "/* block */ 42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			toks := lex(t, tt.src)
			if len(toks) != 2 {
				t.Fatalf("expected 2 tokens (int + EOF), got %d: %v", len(toks), toks)
			}
			if toks[0].Type != tokens.IntLiteral {
				t.Errorf("expected IntLiteral, got %s", toks[0].Type)
			}
		})
	}
}

func TestEOFOnlyForEmptyInput(t *testing.T) {
	t.Parallel()

	toks := lex(t, "")
	if len(toks) != 1 {
		t.Fatalf("expected 1 token (EOF), got %d", len(toks))
	}
	if toks[0].Type != tokens.EOF {
		t.Errorf("expected EOF, got %s", toks[0].Type)
	}
}

func TestEOFPositionMatchesLastToken(t *testing.T) {
	t.Parallel()

	toks := lex(t, "foo bar")
	if len(toks) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(toks))
	}
	lastTok := toks[1]
	eof := toks[2]
	if eof.Ln != lastTok.Ln || eof.Col != lastTok.Col {
		t.Errorf("EOF position (%d:%d) should match last token (%d:%d)",
			eof.Ln, eof.Col, lastTok.Ln, lastTok.Col)
	}
}

func TestMultipleTokenSequence(t *testing.T) {
	t.Parallel()

	toks := lex(t, "x := 42")
	if len(toks) != 4 {
		t.Fatalf("expected 4 tokens, got %d: %v", len(toks), toks)
	}
	if toks[0].Type != tokens.Identifier || toks[0].Literal != "x" {
		t.Errorf("token 0: expected Identifier 'x', got %s %q", toks[0].Type, toks[0].Literal)
	}
	if toks[1].Type != tokens.Declaration {
		t.Errorf("token 1: expected Declaration, got %s", toks[1].Type)
	}
	if toks[2].Type != tokens.IntLiteral || toks[2].Literal != "42" {
		t.Errorf("token 2: expected IntLiteral '42', got %s %q", toks[2].Type, toks[2].Literal)
	}
	if toks[3].Type != tokens.EOF {
		t.Errorf("token 3: expected EOF, got %s", toks[3].Type)
	}
}

func TestDeclarationSequence(t *testing.T) {
	t.Parallel()

	toks := lex(t, `dyn val : utf8 = "hello"`)
	if len(toks) != 7 {
		t.Fatalf("expected 7 tokens, got %d: %v", len(toks), toks)
	}

	expected := []tokens.Type{
		tokens.Dynamic, tokens.Identifier, tokens.Colon,
		tokens.UTF8, tokens.Assign, tokens.StringLiteral, tokens.EOF,
	}
	for i, exp := range expected {
		if toks[i].Type != exp {
			t.Errorf("token %d: expected %s, got %s", i, exp, toks[i].Type)
		}
	}
}

func TestProcDeclarationSequence(t *testing.T) {
	t.Parallel()

	toks := lex(t, "main : proc() = {}")
	if len(toks) != 9 {
		t.Fatalf("expected 9 tokens, got %d: %v", len(toks), toks)
	}

	expected := []tokens.Type{
		tokens.Identifier, tokens.Colon, tokens.Procedure,
		tokens.LParen, tokens.RParen, tokens.Assign,
		tokens.LBrace, tokens.RBrace, tokens.EOF,
	}
	for i, exp := range expected {
		if toks[i].Type != exp {
			t.Errorf("token %d: expected %s, got %s", i, exp, toks[i].Type)
		}
	}
}

func TestContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	l := NewLexer(strings.NewReader("x := 42"))
	toks, err := l.Parse(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(toks) == 0 {
		t.Fatal("expected at least EOF token")
	}
	if toks[len(toks)-1].Type != tokens.EOF {
		t.Errorf("expected last token to be EOF, got %s", toks[len(toks)-1].Type)
	}
}

func TestLineAndColumnTracking(t *testing.T) {
	t.Parallel()

	toks := lex(t, "x\ny")
	if len(toks) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(toks))
	}
	if toks[0].Ln != 1 || toks[0].Col != 1 {
		t.Errorf("'x' at (%d:%d), expected (1:1)", toks[0].Ln, toks[0].Col)
	}
	if toks[1].Ln != 2 || toks[1].Col != 1 {
		t.Errorf("'y' at (%d:%d), expected (2:1)", toks[1].Ln, toks[1].Col)
	}
}

func TestMultiLineProgram(t *testing.T) {
	t.Parallel()

	src := `package main

main : proc() = {
	@print("hello")
}`

	toks := lex(t, src)

	if toks[0].Type != tokens.Package {
		t.Errorf("expected Package, got %s", toks[0].Type)
	}
	if toks[1].Type != tokens.Identifier || toks[1].Literal != "main" {
		t.Errorf("expected Identifier 'main', got %s %q", toks[1].Type, toks[1].Literal)
	}

	var foundPrint bool
	for _, tok := range toks {
		if tok.Type == tokens.Builtin && tok.Literal == "print" {
			foundPrint = true
			break
		}
	}
	if !foundPrint {
		t.Error("expected to find @print builtin token")
	}

	if toks[len(toks)-1].Type != tokens.EOF {
		t.Errorf("expected EOF as last token, got %s", toks[len(toks)-1].Type)
	}
}
