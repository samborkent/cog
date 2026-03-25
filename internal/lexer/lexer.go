package lexer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/scanner"

	"github.com/samborkent/cog/internal/tokens"
)

type Lexer struct {
	scan scanner.Scanner
}

func NewLexer(r io.Reader) *Lexer {
	var s scanner.Scanner
	s.Init(r)
	s.Mode = (scanner.GoTokens | scanner.ScanInts) &^ scanner.SkipComments

	return &Lexer{
		scan: s,
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

		txt := s.TokenText()

		if s.ErrorCount > 0 {
			errs = append(errs, fmt.Errorf("\tln %d, col %d: scanner error: %s", s.Line, s.Column, txt))
			continue
		}

		//nolint:gosec // G115: integer overflow conversion
		t := tokens.Token{
			Ln:  uint32(s.Line),
			Col: uint16(s.Column),
		}

		tokenType, ok := tokens.Runes[tok]
		if ok {
			switch tokenType {
			case tokens.Assign:
				if s.Peek() == '=' {
					t.Type = tokens.Equal
					s.Next()
				}
			case tokens.BitAnd:
				if s.Peek() == '&' {
					t.Type = tokens.And
					s.Next()
				}
			case tokens.Builtin:
				t.Type = tokens.Builtin
				_ = s.Scan()
				t.Literal = s.TokenText()
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
			case tokens.Pipe:
				if s.Peek() == '|' {
					t.Type = tokens.Or
					s.Next()
				}
			}

			if t.Type == 0 {
				t.Type = tokenType
			}

			toks = append(toks, t)
			continue
		}

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
