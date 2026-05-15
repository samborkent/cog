package lexer

import (
	"strings"
	"testing"

	"github.com/samborkent/cog/internal/tokens"
)

func lex(t *testing.T, src string) []tokens.Token {
	t.Helper()

	l := New(strings.NewReader(src), uint32(len(src)), false)

	var toks []tokens.Token
	for {
		tok := l.This()
		toks = append(toks, tok)
		if tok.Type == tokens.EOF {
			break
		}
		l.Step()
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

func TestCommentsArePreserved(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		src     string
		comment string
	}{
		{"line_comment", "// this is a comment\n42", "// this is a comment"},
		{"block_comment", "/* block */ 42", "/* block */"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			toks := lex(t, tt.src)
			if len(toks) != 3 {
				t.Fatalf("expected 3 tokens (comment + int + EOF), got %d: %v", len(toks), toks)
			}

			if toks[0].Type != tokens.Comment {
				t.Errorf("expected Comment, got %s", toks[0].Type)
			}

			if toks[0].Literal != tt.comment {
				t.Errorf("expected comment literal %q, got %q", tt.comment, toks[0].Literal)
			}

			if toks[1].Type != tokens.IntLiteral {
				t.Errorf("expected IntLiteral, got %s", toks[1].Type)
			}
		})
	}
}

func TestInlineComment(t *testing.T) {
	t.Parallel()

	toks := lex(t, "42 // trailing")
	if len(toks) != 3 {
		t.Fatalf("expected 3 tokens (int + comment + EOF), got %d: %v", len(toks), toks)
	}

	if toks[0].Type != tokens.IntLiteral {
		t.Errorf("expected IntLiteral, got %s", toks[0].Type)
	}

	if toks[1].Type != tokens.Comment {
		t.Errorf("expected Comment, got %s", toks[1].Type)
	}

	if toks[1].Literal != "// trailing" {
		t.Errorf("expected literal %q, got %q", "// trailing", toks[1].Literal)
	}
}

func TestMultiLineComment(t *testing.T) {
	t.Parallel()

	toks := lex(t, "/* multi\nline\ncomment */ 42")
	if len(toks) != 3 {
		t.Fatalf("expected 3 tokens (comment + int + EOF), got %d: %v", len(toks), toks)
	}

	if toks[0].Type != tokens.Comment {
		t.Errorf("expected Comment, got %s", toks[0].Type)
	}

	if toks[0].Literal != "/* multi\nline\ncomment */" {
		t.Errorf("expected multi-line comment literal, got %q", toks[0].Literal)
	}

	if toks[1].Type != tokens.IntLiteral {
		t.Errorf("expected IntLiteral, got %s", toks[1].Type)
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
	if eof.Ln < lastTok.Ln || (eof.Ln == lastTok.Ln && eof.Col < lastTok.Col) {
		t.Errorf("EOF position (%d:%d) should not be before last token (%d:%d)",
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
		t.Errorf("expected last token to be EOF, got %s", toks[len(toks)-1].Type)
	}
}

func TestStepMatchesParse(t *testing.T) {
	t.Parallel()

	src := "x := 42 // trailing\n@print(\"ok\")"

	fromLex := lex(t, src)

	l := New(strings.NewReader(src), uint32(len(src)), false)
	var fromStep []tokens.Token

	for {
		tok := l.This()
		fromStep = append(fromStep, tok)
		if tok.Type == tokens.EOF {
			break
		}
		l.Step()
	}

	if len(fromStep) != len(fromLex) {
		t.Fatalf("expected %d tokens from Step, got %d", len(fromLex), len(fromStep))
	}

	for i := range fromLex {
		if fromStep[i] != fromLex[i] {
			t.Fatalf("token %d mismatch: Step=%+v Lex=%+v", i, fromStep[i], fromLex[i])
		}
	}
}

func TestStepScannerErrorTracking(t *testing.T) {
	t.Parallel()

	src := "\"unterminated\nx := 1"

	// Unterminated string triggers a scanner error; Step should still continue with later tokens.
	l := New(strings.NewReader(src), uint32(len(src)), false)

	gotTypes := make([]tokens.Type, 0, 8)
	for {
		tok := l.This()
		gotTypes = append(gotTypes, tok.Type)
		if tok.Type == tokens.EOF {
			break
		}
		l.Step()
	}

	if len(l.errs) != 1 {
		t.Fatalf("expected exactly 1 scanner error, got %d (%v)", len(l.errs), l.errs)
	}

	if err := l.Err(); err == nil {
		t.Fatal("Err() should return non-nil after scanner error")
	}

	expected := []tokens.Type{tokens.Identifier, tokens.Declaration, tokens.IntLiteral, tokens.EOF}
	if len(gotTypes) != len(expected) {
		t.Fatalf("expected %d tokens, got %d (%v)", len(expected), len(gotTypes), gotTypes)
	}

	for i := range expected {
		if gotTypes[i] != expected[i] {
			t.Fatalf("token %d: expected %s, got %s", i, expected[i], gotTypes[i])
		}
	}
}

func TestPeekDoesNotConsume(t *testing.T) {
	t.Parallel()

	src := "x := 42"

	l := New(strings.NewReader(src), uint32(len(src)), false)

	peek1 := l.Peek(1)
	peek2 := l.Peek(2)
	peek1Again := l.Peek(1)

	if peek1.Type != tokens.Declaration {
		t.Fatalf("peek(1): expected Declaration, got %s %q", peek1.Type, peek1.Literal)
	}

	if peek2.Type != tokens.IntLiteral {
		t.Fatalf("peek(2): expected IntLiteral, got %s", peek2.Type)
	}

	if peek1Again != peek1 {
		t.Fatalf("peek(1) should be stable, got %+v then %+v", peek1, peek1Again)
	}

	l.Step()
	current := l.This()
	if current != peek1 {
		t.Fatalf("step should match prior peek(1): current=%+v peek=%+v", current, peek1)
	}
}

func TestPeekBeyondEOFReturnsEOF(t *testing.T) {
	t.Parallel()

	src := "x"

	l := New(strings.NewReader(src), uint32(len(src)), false)

	if tok := l.Peek(0); tok.Type != tokens.Identifier {
		t.Fatalf("peek(0): expected Identifier, got %s", tok.Type)
	}

	if tok := l.Peek(1); tok.Type != tokens.EOF {
		t.Fatalf("peek(1): expected EOF, got %s", tok.Type)
	}

	if tok := l.Peek(2); tok != (tokens.Token{}) {
		t.Fatalf("peek(2): expected zero token, got %+v", tok)
	}

	l.Step()
	if tok := l.This(); tok.Type != tokens.EOF {
		t.Fatalf("next 1: expected EOF, got %s", tok.Type)
	}

	l.Step()
	if tok := l.This(); tok.Type != tokens.EOF {
		t.Fatalf("next 2: expected EOF (stay), got %s", tok.Type)
	}
}

func TestPeekCurrentAndHistory(t *testing.T) {
	t.Parallel()

	src := "x := 42"

	l := New(strings.NewReader(src), uint32(len(src)), false)

	// After New: primed to first token
	if tok := l.This(); tok.Type != tokens.Identifier {
		t.Fatalf("peek(0) after New: expected Identifier, got %+v", tok)
	}

	if tok := l.Peek(-1); tok != (tokens.Token{}) {
		t.Fatalf("peek(-1) after New: expected zero token, got %+v", tok)
	}

	l.Step()
	if tok := l.This(); tok.Type != tokens.Declaration {
		t.Fatalf("after step 1: expected Declaration, got %s", tok.Type)
	}

	if tok := l.Peek(-1); tok.Type != tokens.Identifier {
		t.Fatalf("peek(-1) after step 1: expected Identifier, got %s", tok.Type)
	}

	if tok := l.Peek(-2); tok != (tokens.Token{}) {
		t.Fatalf("peek(-2) after step 1: expected zero token, got %+v", tok)
	}

	l.Step()
	if tok := l.This(); tok.Type != tokens.IntLiteral {
		t.Fatalf("after step 2: expected IntLiteral, got %s", tok.Type)
	}

	if tok := l.Peek(-1); tok.Type != tokens.Declaration {
		t.Fatalf("peek(-1) after step 2: expected Declaration, got %s", tok.Type)
	}

	if tok := l.Peek(-2); tok.Type != tokens.Identifier {
		t.Fatalf("peek(-2) after step 2: expected Identifier, got %s", tok.Type)
	}
}

func TestStepAdvancesCurrentToken(t *testing.T) {
	t.Parallel()

	src := "x := 42"

	l := New(strings.NewReader(src), uint32(len(src)), false)

	if tok := l.This(); tok.Type != tokens.Identifier {
		t.Fatalf("peek(0) after New: expected Identifier, got %+v", tok)
	}

	expected := []tokens.Type{
		tokens.Declaration,
		tokens.IntLiteral,
		tokens.EOF,
		tokens.EOF,
	}

	for i, want := range expected {
		l.Step()

		if got := l.This().Type; got != want {
			t.Fatalf("after step %d: expected %s, got %s", i+1, want, got)
		}
	}
}

func TestSkipBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      string
		wantType tokens.Type
		wantLit  string
	}{
		{
			name:     "simple body",
			src:      `{ x := 42 } after`,
			wantType: tokens.Identifier,
			wantLit:  "after",
		},
		{
			name:     "nested braces",
			src:      `{ if true { nested } } after`,
			wantType: tokens.Identifier,
			wantLit:  "after",
		},
		{
			name:     "body with strings containing braces",
			src:      `{ x := "{}}" } after`,
			wantType: tokens.Identifier,
			wantLit:  "after",
		},
		{
			name:     "body at EOF",
			src:      `{ x := 1 }`,
			wantType: tokens.EOF,
		},
		{
			name:     "deeply nested",
			src:      `{ { { { } } } } done`,
			wantType: tokens.Identifier,
			wantLit:  "done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := New(strings.NewReader(tt.src), uint32(len(tt.src)), false)

			if l.This().Type != tokens.LBrace {
				t.Fatalf("expected LBrace, got %s", l.This().Type)
			}

			l.SkipBody()

			got := l.This()
			if got.Type != tt.wantType {
				t.Fatalf("after SkipBody: expected %s, got %s", tt.wantType, got.Type)
			}

			if tt.wantLit != "" && got.Literal != tt.wantLit {
				t.Fatalf("after SkipBody: expected literal %q, got %q", tt.wantLit, got.Literal)
			}
		})
	}
}

func TestSkipBodyWithLookahead(t *testing.T) {
	t.Parallel()

	// Simulate lookahead before SkipBody (Peek fills ring buffer ahead).
	src := `{ a } after`

	l := New(strings.NewReader(src), uint32(len(src)), false)

	// Fill lookahead slots.
	_ = l.Peek(1) // 'a'
	_ = l.Peek(2) // '}'

	l.SkipBody()

	got := l.This()
	if got.Type != tokens.Identifier || got.Literal != "after" {
		t.Fatalf("expected Identifier 'after', got %s %q", got.Type, got.Literal)
	}
}
