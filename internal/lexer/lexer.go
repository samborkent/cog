package lexer

import (
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"slices"
	"strings"
	"text/scanner"

	"github.com/samborkent/cog/internal/tokens"
)

const windowSize = 7

// tokenOffset pairs a token with the byte offset of its start in the source.
type tokenOffset struct {
	tokens.Token
	offset uint32
}

type Lexer struct {
	scan    *scanner.Scanner
	src     io.ReadSeeker
	window  [windowSize]tokenOffset // ring buffer: offsets [-3, -2, -1, 0, +1, +2, +3] from cursor
	errs    []error
	Len     uint32 // total length of the input in bytes
	cursor  uint8  // window slot of the current token
	scanErr bool   // set by s.Error callback, cleared after each scanNext iteration
	debug   bool
	baseLn  uint32 // line offset applied to tokens after SeekTo (0 for top-level parse)
}

func setupScanner(s *scanner.Scanner, r io.ReadSeeker) {
	s.Init(r)
	// Tokenize identifiers, floats, characters, (multi-line) strings, ints & comments.
	s.Mode = (scanner.GoTokens | scanner.ScanInts) &^ scanner.SkipComments
	// Tokenize new lines.
	s.Whitespace = 1<<'\t' | 1<<'\r' | 1<<' '
}

func New(r io.ReadSeeker, len uint32, debug bool) *Lexer {
	s := new(scanner.Scanner)
	setupScanner(s, r)

	l := &Lexer{
		scan:  s,
		src:   r,
		errs:  make([]error, 0),
		Len:   len,
		debug: debug,
	}

	closer, ok := r.(io.Closer)
	if ok {
		// Close the reader when the lexer is garbage collected, to ensure resources are freed even if the caller forgets.
		runtime.AddCleanup(l, func(_ int) {
			_ = closer.Close()
		}, 0)
	}

	s.Error = func(s *scanner.Scanner, msg string) {
		l.scanErr = true
		l.errs = append(l.errs, fmt.Errorf("\tln %d, col %d: scanner error: %s", s.Line, s.Column, msg))
	}

	l.Step() // prime current token to first lexical token

	return l
}

func (l *Lexer) Err() error {
	if err := errors.Join(l.errs...); err != nil {
		return fmt.Errorf("tokenization error(s):\n%w", err)
	}

	return nil
}

func (l *Lexer) Reset() {
	_, _ = l.src.Seek(0, io.SeekStart)
	setupScanner(l.scan, l.src)
	l.cursor = 0
	l.errs = l.errs[:0]
	l.scanErr = false
	l.window = [windowSize]tokenOffset{}
	l.Step()
}

// SeekTo seeks the lexer to the given byte offset in the source, clears the
// ring buffer, and primes with the token at that position. Used for deferred
// body parsing — seek back to a '{' captured earlier via Offset().
// baseLine is the global line number at the seek offset. Token line numbers
// are reported as baseLine - 1 + scanner.Line, so the first token at the seek
// position (absolute line baseLine) reports line baseLine.
func (l *Lexer) SeekTo(offset uint32, baseLine uint32) {
	_, _ = l.src.Seek(int64(offset), io.SeekStart)
	setupScanner(l.scan, l.src)
	l.cursor = 0
	l.scanErr = false
	l.window = [windowSize]tokenOffset{}
	l.baseLn = baseLine - 1
	l.Step()
}

// SkipBody skips over a brace-delimited body using the raw scanner without
// producing full Token objects. Must be called when This() is the opening '{'.
// After return, This() is the first token after the closing '}'.
func (l *Lexer) SkipBody() {
	depth := 1
	s := l.scan

	// Drain any cached lookahead tokens the scanner already produced.
	for i := 1; i <= 3; i++ {
		idx := l.windowIndex(i)
		tok := l.window[idx]
		if tok == (tokenOffset{}) || tok.Type == tokens.EOF {
			break
		}

		switch tok.Type {
		case tokens.LBrace:
			depth++
		case tokens.RBrace:
			depth--
			if depth == 0 {
				goto done
			}
		}
	}

	// Continue with raw scanner — only inspect braces.
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		if l.scanErr {
			l.scanErr = false
			continue
		}

		switch tok {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				goto done
			}
		}
	}

done:
	// Clear ring buffer and prime with the next token after '}'.
	l.window = [windowSize]tokenOffset{}
	l.cursor = 0
	l.window[0] = l.scanNext()
}

// Step advances the lexer cursor by one token.
// The current token after stepping can be read with Peek(0).
func (l *Lexer) Step() {
	if l.window[l.cursor].Type == tokens.EOF {
		return
	}

	// Read directly from the +1 slot; call scanNext only if it's unfilled.
	aheadIdx := l.windowIndex(1)
	to := l.window[aheadIdx]
	if to == (tokenOffset{}) {
		to = l.scanNext()
	}

	l.cursor = aheadIdx
	l.window[l.cursor] = to

	// Clear the stale +3 slot exposed by the cursor rotation.
	l.window[l.windowIndex(3)] = tokenOffset{}

	if l.debug {
		from := to.Type.String()
		if slices.Contains([]tokens.Type{
			tokens.Identifier, tokens.StringLiteral, tokens.IntLiteral, tokens.FloatLiteral,
		}, to.Type) {
			from = to.Literal
		}

		next := l.Peek(1).Type.String()
		if slices.Contains([]tokens.Type{
			tokens.Identifier, tokens.StringLiteral, tokens.IntLiteral, tokens.FloatLiteral,
		}, l.Peek(1).Type) {
			next = l.Peek(1).Literal
		}

		_, _ = fmt.Printf("DEBUG: ln %d, col %d:\tfrom %q,\tto %q\n",
			to.Ln, to.Col, from, next)
	}
}

// This is shorthand for getting the current token.
func (l *Lexer) This() tokens.Token {
	return l.window[l.cursor].Token
}

// Offset returns the byte offset of the current token's start in the source.
func (l *Lexer) Offset() uint32 {
	return l.window[l.cursor].offset
}

// Peek returns a token at offset n from current (0 = current, ±1/±2/±3 = look-ahead/behind).
// Returns tokens.Token{} for out-of-range or unavailable history.
func (l *Lexer) Peek(n int) tokens.Token {
	switch n {
	case 0:
		return l.window[l.cursor].Token
	case -1, -2, -3:
		return l.window[l.windowIndex(n)].Token // zero-value if fewer than |n| tokens consumed
	case 1, 2, 3:
		// Fill lookahead slots lazily.
		for i := 1; i <= n; i++ {
			if i > 1 {
				prev := l.window[l.windowIndex(i-1)]
				if prev == (tokenOffset{}) || prev.Type == tokens.EOF {
					break
				}
			}

			idx := l.windowIndex(i)
			if l.window[idx] != (tokenOffset{}) {
				continue
			}

			l.window[idx] = l.scanNext()

			if l.window[idx].Type == tokens.EOF {
				break
			}
		}

		return l.window[l.windowIndex(n)].Token // zero token if EOF arrived before offset n
	}

	return tokens.Token{}
}

// windowIndex converts a relative offset to a window array index.
func (l *Lexer) windowIndex(offset int) uint8 {
	return uint8((int(l.cursor) + offset + windowSize) % windowSize)
}

func (l *Lexer) scanNext() tokenOffset {
	s := l.scan

	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		// The Error callback sets scanErr; clear and skip the invalid token.
		if l.scanErr {
			l.scanErr = false
			continue
		}

		txt := s.TokenText()

		t := tokens.Token{
			Pos: tokens.Pos{
				Ln:  l.baseLn + uint32(min(s.Line, math.MaxUint32)),
				Col: uint16(min(s.Column, math.MaxUint16)),
			},
		}

		switch tok {
		case scanner.Ident:
			tokenType, ok := tokens.Keywords[txt]
			if ok {
				// cog keyword
				t.Type = tokenType
			} else {
				// identifier
				t.Type = tokens.Identifier
				t.Literal = txt
			}
		case scanner.Comment:
			t.Type = tokens.Comment
			t.Literal = txt
		case scanner.String:
			t.Type = tokens.StringLiteral
			t.Literal = strings.Trim(txt, `"`)
		case scanner.RawString:
			t.Type = tokens.StringLiteral
			t.Literal = strings.Trim(txt, "`")
		case scanner.Int:
			t.Type = tokens.IntLiteral
			t.Literal = txt
		case scanner.Float:
			t.Type = tokens.FloatLiteral
			t.Literal = txt
		case '\n':
			t.Type = tokens.Newline
			t.Literal = "\\n"
		default:
			tokenType, ok := tokens.Runes[tok]
			if !ok {
				l.errs = append(l.errs,
					fmt.Errorf("\tln %d, col %d: unknown token: %s", s.Line, s.Column, txt))
				continue
			}

			switch tokenType {
			case tokens.Assign:
				if s.Peek() == '=' {
					t.Type = tokens.Equal

					s.Next()
				}
			case tokens.Colon:
				switch s.Peek() {
				case '=':
					t.Type = tokens.Declaration

					s.Next()
				}
			case tokens.GT:
				if s.Peek() == '=' {
					t.Type = tokens.GTEqual

					s.Next()
				}
			case tokens.LT:
				if s.Peek() == '=' {
					t.Type = tokens.LTEqual

					s.Next()
				}
			case tokens.Not:
				if s.Peek() == '=' {
					t.Type = tokens.NotEqual

					s.Next()
				}
			case tokens.BitAnd:
				if s.Peek() == '&' {
					t.Type = tokens.And

					s.Next()
				}
			case tokens.Pipe:
				if s.Peek() == '|' {
					t.Type = tokens.Or

					s.Next()
				}
			case tokens.Builtin:
				t.Type = tokens.Builtin
				_ = s.Scan()
				t.Literal = s.TokenText()
			}

			if t.Type == 0 {
				t.Type = tokenType
			}
		}

		return tokenOffset{Token: t, offset: uint32(s.Position.Offset)}
	}

	return tokenOffset{
		Token: tokens.Token{
			Pos: tokens.Pos{
				Ln:  uint32(min(s.Line, math.MaxUint32)),
				Col: uint16(min(s.Column, math.MaxUint16)),
			},
			Type: tokens.EOF,
		},
		offset: uint32(s.Pos().Offset),
	}
}
