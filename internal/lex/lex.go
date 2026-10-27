// Package lex turns a Vulcan source file into Go tokens with byte offsets, and finds the
// signatures and contract clauses of Appendix B.1 in them.
package lex

import (
	"go/scanner"
	"go/token"
	"strings"
)

// Tok is one token. Off and End are byte offsets in the source. A semicolon that the scanner
// inserted at a line end has Lit "\n" and End == Off.
type Tok struct {
	Tok token.Token
	Lit string
	Off int
	End int
}

// Is reports whether t is the identifier name.
func (t Tok) Is(name string) bool { return t.Tok == token.IDENT && t.Lit == name }

// AutoSemi reports whether t is a semicolon that the scanner inserted.
func (t Tok) AutoSemi() bool { return t.Tok == token.SEMICOLON && t.Lit == "\n" }

// Error is a lexical error at a byte offset.
type Error struct {
	Off int
	Msg string
}

// Pad is the number of EOF tokens at the end of a scan, so that a lookahead of a few tokens
// after a truncated input never runs past the slice.
const Pad = 4

// Scan returns the tokens of src, ending with Pad EOF tokens, and the lexical errors.
func Scan(src []byte) ([]Tok, []Error) {
	fset := token.NewFileSet()
	file := fset.AddFile("", -1, len(src))
	var errs []Error
	var s scanner.Scanner
	s.Init(file, src, func(pos token.Position, msg string) {
		errs = append(errs, Error{Off: pos.Offset, Msg: msg})
	}, 0)
	var toks []Tok
	for {
		pos, tok, lit := s.Scan()
		off := file.Offset(pos)
		t := Tok{Tok: tok, Lit: lit, Off: off}
		switch {
		case tok == token.SEMICOLON && lit == "\n":
			t.End = off
		case tok == token.EOF:
			t.End = off
		case lit != "":
			t.End = off + len(lit)
		default:
			t.End = off + len(tok.String())
		}
		// The scanner gives an invalid byte the 3-byte literal U+FFFD.
		t.End = min(t.End, len(src))
		toks = append(toks, t)
		if tok == token.EOF {
			for range Pad - 1 {
				toks = append(toks, t)
			}
			return toks, errs
		}
	}
}

// Match returns the index of the token that closes the bracket at toks[i], or len(toks)-1.
func Match(toks []Tok, i int) int {
	open := toks[i].Tok
	switch open {
	case token.LPAREN, token.LBRACK, token.LBRACE:
	default:
		return i
	}
	depth := 0
	for j := i; j < len(toks); j++ {
		switch toks[j].Tok {
		case token.LPAREN, token.LBRACK, token.LBRACE:
			depth++
		case token.RPAREN, token.RBRACK, token.RBRACE:
			depth--
			if depth == 0 {
				return j // the closer, or a mismatched one, which the Go parser reports
			}
		case token.EOF:
			return j
		}
	}
	return len(toks) - 1
}

// Directives returns the offsets of each //go:<name> comment that starts a line.
func Directives(src []byte, name string) []int {
	fset := token.NewFileSet()
	file := fset.AddFile("", -1, len(src))
	var s scanner.Scanner
	s.Init(file, src, func(token.Position, string) {}, scanner.ScanComments)
	var out []int
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			return out
		}
		if tok != token.COMMENT || !strings.HasPrefix(lit, "//go:"+name) {
			continue
		}
		off := file.Offset(pos)
		start := off
		for start > 0 && (src[start-1] == ' ' || src[start-1] == '\t') {
			start--
		}
		if start == 0 || src[start-1] == '\n' {
			out = append(out, off)
		}
	}
}

// ContractWords are the identifiers that start a contract clause after a signature.
var ContractWords = map[string]bool{"requires": true, "ensures": true}

// IsContractStart reports whether toks[i] starts a contract clause, by the rule of Appendix B.1:
// the word is not followed by "{", ";", ".", or "[", and does not follow a ".".
func IsContractStart(toks []Tok, i int, words map[string]bool) bool {
	t := toks[i]
	if t.Tok != token.IDENT || !words[t.Lit] {
		return false
	}
	if i > 0 && toks[i-1].Tok == token.PERIOD {
		return false
	}
	if i+1 < len(toks) {
		switch toks[i+1].Tok {
		case token.LBRACE, token.SEMICOLON, token.PERIOD, token.LBRACK, token.EOF:
			return false
		}
	}
	return true
}

// SigEnd returns the index of the first token after a signature whose parameter list starts at
// toks[i] (a "("). The result type, if any, is part of the signature.
func SigEnd(toks []Tok, i int, words map[string]bool) int {
	j := Match(toks, i) + 1
	if j >= len(toks) {
		return len(toks) - 1
	}
	if toks[j].Tok == token.LPAREN {
		return Match(toks, j) + 1
	}
	depth := 0
	for ; j < len(toks); j++ {
		t := toks[j]
		if depth == 0 {
			switch {
			case t.Tok == token.LBRACE && !(j > 0 && (toks[j-1].Tok == token.STRUCT || toks[j-1].Tok == token.INTERFACE)):
				return j
			case t.Tok == token.SEMICOLON, t.Tok == token.EOF, t.Tok == token.RBRACE, t.Tok == token.RPAREN:
				return j
			case IsContractStart(toks, j, words):
				return j
			}
		}
		switch t.Tok {
		case token.LPAREN, token.LBRACK, token.LBRACE:
			depth++
		case token.RPAREN, token.RBRACK, token.RBRACE:
			depth--
		}
	}
	return len(toks) - 1
}

// Clause is one contract clause: its word and the offsets of its expression.
type Clause struct {
	Word     string
	WordOff  int
	Off, End int // the expression
}

// Clauses reads the contract clauses that start at toks[k], the token after a signature (or
// after a type, for "invariant"). It returns the clauses and the index of the token after them:
// the body's "{", or the ";" that ends the declaration. Line-end semicolons before a clause or
// before the body are skipped, as Appendix B.1 says.
func Clauses(toks []Tok, k int, words map[string]bool) ([]Clause, int) {
	var out []Clause
	for {
		if toks[k].AutoSemi() && k+1 < len(toks) && IsContractStart(toks, k+1, words) && len(words) > 0 {
			k++
		}
		if !IsContractStart(toks, k, words) {
			return out, k
		}
		c := Clause{Word: toks[k].Lit, WordOff: toks[k].Off}
		k++
		c.Off = toks[k].Off
		depth := 0
		end := k
		for ; k < len(toks); k++ {
			t := toks[k]
			if t.Tok == token.EOF {
				break
			}
			if depth == 0 {
				if t.Tok == token.LBRACE || t.Tok == token.RBRACE {
					break
				}
				if t.Tok == token.SEMICOLON {
					break
				}
			}
			switch t.Tok {
			case token.LPAREN, token.LBRACK, token.LBRACE:
				depth++
			case token.RPAREN, token.RBRACK, token.RBRACE:
				depth--
			}
			end = k
		}
		c.End = toks[end].End
		out = append(out, c)
		// A line end before the body's "{" is skipped.
		if toks[k].AutoSemi() && k+1 < len(toks) && toks[k+1].Tok == token.LBRACE {
			k++
		}
	}
}

// Lines computes line starts once, for fast conversion.
type Lines struct{ starts []int }

// NewLines indexes the line starts of src.
func NewLines(src []byte) *Lines {
	l := &Lines{starts: []int{0}}
	for i, b := range src {
		if b == '\n' {
			l.starts = append(l.starts, i+1)
		}
	}
	return l
}

// Pos converts a byte offset to a 1-based line and column.
func (l *Lines) Pos(off int) (int, int) {
	lo, hi := 0, len(l.starts)
	for hi-lo > 1 {
		m := (lo + hi) / 2
		if l.starts[m] <= off {
			lo = m
		} else {
			hi = m
		}
	}
	return lo + 1, off - l.starts[lo] + 1
}
