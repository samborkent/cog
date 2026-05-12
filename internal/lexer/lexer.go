package lexer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"math"
	"strings"
	"text/scanner"

	"github.com/samborkent/cog/internal/tokens"
)

type Lexer struct {
	scan           *scanner.Scanner
	window         [5]tokens.Token
	errs           []error
	scanErrorCount int
	index          int
	cursor         uint8
}

func New(r io.Reader) *Lexer {
	s := new(scanner.Scanner)
	s.Init(r)
	s.Mode = (scanner.GoTokens | scanner.ScanInts) &^ scanner.SkipComments

	return &Lexer{
		scan:  s,
		errs:  make([]error, 0),
		index: -1, // Init at -1, so first Next() call increments to 0.
	}
}

func (l *Lexer) Range() iter.Seq2[int, tokens.Token] {
	return func(yield func(int, tokens.Token) bool) {
		for {
			tok := l.Next()

			if tok.Type == tokens.EOF {
				return
			}

			if !yield(l.index, tok) {
				return
			}
		}
	}
}

func (l *Lexer) Next() tokens.Token {
	if l.window[l.cursor].Type == tokens.EOF {
		return l.window[l.cursor]
	}

	l.index++

	tok := l.Peek(1)
	if tok == (tokens.Token{}) {
		tok = l.scanNext()
	}

	l.cursor = l.windowIndex(1)
	l.window[l.cursor] = tok

	// Clear the newly exposed farthest lookahead slot.
	l.window[l.windowIndex(2)] = tokens.Token{}

	return tok
}

func (l *Lexer) Peek(n int) tokens.Token {
	if n < -2 || n > 2 {
		return tokens.Token{}
	}

	switch n {
	case 0:
		return l.window[l.cursor]
	case -1, -2:
		return l.window[l.windowIndex(n)]
	case 1, 2:
		for i := 1; i <= n; i++ {
			idx := l.windowIndex(i)
			if l.window[idx] != (tokens.Token{}) {
				continue
			}

			tok := l.scanNext()
			l.window[idx] = tok

			if tok.Type == tokens.EOF {
				break
			}
		}

		tok := l.window[l.windowIndex(n)]
		if tok != (tokens.Token{}) {
			return tok
		}

		for i := n - 1; i >= 1; i-- {
			tok = l.window[l.windowIndex(i)]
			if tok != (tokens.Token{}) {
				return tok
			}
		}

		return tokens.Token{}
	default:
		return tokens.Token{}
	}
}

func (l *Lexer) windowIndex(offset int) uint8 {
	const windowSize = 5

	idx := (int(l.cursor) + offset + windowSize) % windowSize

	return uint8(idx)
}

func (l *Lexer) scanNext() tokens.Token {
	s := l.scan

	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		txt := s.TokenText()

		if s.ErrorCount > l.scanErrorCount {
			l.scanErrorCount = s.ErrorCount
			l.errs = append(l.errs, fmt.Errorf("\tln %d, col %d: scanner error: %s", s.Line, s.Column, txt))
			continue
		}

		t := tokens.Token{
			Ln:  uint32(min(s.Line, math.MaxUint32)),
			Col: uint16(min(s.Column, math.MaxUint16)),
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

		return t
	}

	return tokens.Token{
		Type: tokens.EOF,
		Ln:   uint32(min(s.Line, math.MaxUint32)),
		Col:  uint16(min(s.Column, math.MaxUint16)),
	}
}

func (l *Lexer) Parse(ctx context.Context) ([]tokens.Token, error) {
	s := l.scan

	var errs []error

	// TODO: determine appropriate pre-allocation size, or guess number of tokens based on file size.
	toks := make([]tokens.Token, 0, 1024)

	var (
		ln  uint32
		col uint16
	)

	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		if ctx.Err() != nil {
			break
		}

		if s.ErrorCount > 0 {
			errs = append(errs, fmt.Errorf("\tln %d, col %d: scanner error: %s", s.Line, s.Column, s.TokenText()))
			continue
		}

		t := tokens.Token{
			Ln:  uint32(min(s.Line, math.MaxUint32)),
			Col: uint16(min(s.Column, math.MaxUint16)),
		}

		tokenType, ok := tokens.Runes[tok]
		if ok {
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

			toks = append(toks, t)

			continue
		}

		txt := s.TokenText()

		switch tok {
		case scanner.Comment:
			t.Type = tokens.Comment
			t.Literal = txt
		case scanner.Int:
			t.Type = tokens.IntLiteral
			t.Literal = txt
		case scanner.Float:
			t.Type = tokens.FloatLiteral
			t.Literal = txt
		case scanner.String:
			t.Type = tokens.StringLiteral
			t.Literal = strings.Trim(txt, `"`)
		case scanner.RawString:
			t.Type = tokens.StringLiteral
			t.Literal = strings.Trim(txt, "`")
		case scanner.Ident:
			tokenType, ok := tokens.Keywords[txt]
			if ok {
				t.Type = tokenType
			} else {
				t.Type = tokens.Identifier
				t.Literal = txt
			}
		default:
			errs = append(errs, fmt.Errorf("\tln %d, col %d: unknown token: %s", s.Line, s.Column, txt))
			continue
		}

		toks = append(toks, t)
		ln = t.Ln
		col = t.Col
	}

	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("tokenization error:\n%w", err)
	}

	eof := tokens.Token{
		Type: tokens.EOF,
		Ln:   ln,
		Col:  col,
	}

	return append(toks, eof), nil
}
