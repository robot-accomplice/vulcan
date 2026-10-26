// Package blank turns a .vulcan code file into Go source (spec §11.1, Appendix B.1). It replaces
// the Vulcan-only text with spaces, so that the lines and the columns of every other token stay
// the same, and it records what it removed.
package blank

import (
	"bytes"
	"go/token"
	"regexp"

	"github.com/robot-accomplice/vulcan/internal/lex"
)

// Clause is one contract clause that the blanker removed.
type Clause struct {
	Word     string // "requires", "ensures", or "invariant"
	WordOff  int
	Off, End int // the expression, in the source
	Text     string
}

// Func is a function or a method with contract clauses.
type Func struct {
	Name    string
	Recv    string // the receiver's type name, or "" for a function
	NameOff int
	BodyOff int // the offset of the body's "{" in the Go output
	Clauses []Clause
	HasBody bool
}

// Type is a struct type with an invariant.
type Type struct {
	Name    string
	NameOff int
	Clauses []Clause
}

// Deferred is a construct that the release does not support.
type Deferred struct {
	What, Section string
	Off           int
}

// Result is the outcome of blanking one file.
type Result struct {
	Go        []byte
	Funcs     []*Func
	Types     []*Type
	Deferred  []Deferred
	Linknames []int // offsets of //go:linkname directives
	LexErrors []lex.Error
	Misplaced []Clause // invariant in a parenthesized type group (K0513)
}

type edit struct {
	off, end int
	text     string
}

var funcWords = lex.ContractWords
var invWords = map[string]bool{"invariant": true}

var linkname = regexp.MustCompile(`(?m)^[ \t]*//go:linkname\b`)

// File blanks src.
func File(src []byte) *Result {
	toks, errs := lex.Scan(src)
	r := &Result{LexErrors: errs}
	for _, m := range linkname.FindAllIndex(src, -1) {
		r.Linknames = append(r.Linknames, m[0]+bytes.Index(src[m[0]:m[1]], []byte("//")))
	}
	var edits []edit
	blankRange := func(off, end int) {
		edits = append(edits, edit{off: off, end: end})
	}
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		switch {
		case t.Tok == token.EOF:
			i = len(toks)
		case t.Is("pure") && toks[i+1].Tok == token.FUNC:
			r.Deferred = append(r.Deferred, Deferred{What: "pure functions", Section: "§7.1", Off: t.Off})
			blankRange(t.Off, t.End)
		case t.Is("handle") && toks[i+1].Tok == token.IDENT && toks[i+2].Tok == token.LPAREN && (i == 0 || toks[i-1].Tok == token.SEMICOLON):
			r.Deferred = append(r.Deferred, Deferred{What: "handle (topic handlers)", Section: "§5.5", Off: t.Off})
			end := skipDecl(toks, i)
			edits = append(edits, edit{off: t.Off, end: toks[end].End})
			i = end
		case t.Is("idempotent") && toks[i+1].Is("handle"):
			r.Deferred = append(r.Deferred, Deferred{What: "idempotent handle (topic handlers)", Section: "§5.5", Off: t.Off})
			end := skipDecl(toks, i)
			edits = append(edits, edit{off: t.Off, end: toks[end].End})
			i = end
		case t.Tok == token.FUNC:
			i = funcDecl(src, toks, i, r, &edits)
		case t.Tok == token.TYPE:
			start := i
			i = typeDecl(src, toks, i, r, &edits)
			limits(toks, start, i, r, &edits)
		case t.Tok == token.LBRACE || t.Tok == token.LPAREN || t.Tok == token.LBRACK:
			i = lex.Match(toks, i)
		}
	}
	r.Go = apply(src, edits)
	return r
}

// skipDecl skips a declaration up to and including its body.
func skipDecl(toks []lex.Tok, i int) int {
	for ; i < len(toks); i++ {
		if toks[i].Tok == token.LBRACE {
			return lex.Match(toks, i)
		}
		if toks[i].Tok == token.EOF {
			return i
		}
	}
	return i
}

func funcDecl(src []byte, toks []lex.Tok, i int, r *Result, edits *[]edit) int {
	f := &Func{}
	j := i + 1
	if toks[j].Tok == token.LPAREN { // receiver
		end := lex.Match(toks, j)
		for k := j + 1; k < end; k++ {
			if toks[k].Tok == token.IDENT && (toks[k+1].Tok == token.RPAREN || toks[k+1].Tok == token.LBRACK) {
				f.Recv = toks[k].Lit
			}
		}
		j = end + 1
	}
	if toks[j].Tok != token.IDENT {
		return j
	}
	f.Name, f.NameOff = toks[j].Lit, toks[j].Off
	j++
	if toks[j].Tok == token.LBRACK {
		j = lex.Match(toks, j) + 1
	}
	if toks[j].Tok != token.LPAREN {
		return j
	}
	sigEnd := lex.SigEnd(toks, j, funcWords)
	clauses, k := lex.Clauses(toks, sigEnd, funcWords)
	if len(clauses) > 0 {
		for _, c := range clauses {
			f.Clauses = append(f.Clauses, Clause{Word: c.Word, WordOff: c.WordOff, Off: c.Off, End: c.End, Text: string(src[c.Off:c.End])})
			*edits = append(*edits, edit{off: c.WordOff, end: c.End})
		}
		r.Funcs = append(r.Funcs, f)
		if toks[k].Tok == token.LBRACE {
			f.HasBody = true
			// Move the body's "{" to the end of the signature line.
			// When the "{" is already on that line, it stays, so no column moves.
			sigLast := toks[sigEnd-1].End
			if bytes.IndexByte(src[sigLast:toks[k].Off], '\n') >= 0 {
				*edits = append(*edits, edit{off: sigLast, end: sigLast, text: " {"})
				*edits = append(*edits, edit{off: toks[k].Off, end: toks[k].End})
			}
		}
	}
	if toks[k].Tok == token.LBRACE {
		return lex.Match(toks, k)
	}
	return k
}

func typeDecl(src []byte, toks []lex.Tok, i int, r *Result, edits *[]edit) int {
	j := i + 1
	if toks[j].Tok == token.LPAREN {
		end := lex.Match(toks, j)
		for k := j; k < end; k++ {
			if toks[k].Is("invariant") && k > 0 && toks[k-1].AutoSemi() {
				r.Misplaced = append(r.Misplaced, Clause{Word: "invariant", WordOff: toks[k].Off})
			}
		}
		return end
	}
	if toks[j].Tok != token.IDENT {
		return j
	}
	ty := &Type{Name: toks[j].Lit, NameOff: toks[j].Off}
	// The type ends at a semicolon at depth 0.
	depth := 0
	k := j + 1
	for ; k < len(toks); k++ {
		t := toks[k]
		if depth == 0 && (t.Tok == token.SEMICOLON || t.Tok == token.EOF) {
			break
		}
		switch t.Tok {
		case token.LPAREN, token.LBRACK, token.LBRACE:
			depth++
		case token.RPAREN, token.RBRACK, token.RBRACE:
			depth--
		}
	}
	if k+1 < len(toks) && toks[k].AutoSemi() && toks[k+1].Is("invariant") {
		clauses, end := invClauses(toks, k+1)
		for _, c := range clauses {
			ty.Clauses = append(ty.Clauses, Clause{Word: c.Word, WordOff: c.WordOff, Off: c.Off, End: c.End, Text: string(src[c.Off:c.End])})
			*edits = append(*edits, edit{off: c.WordOff, end: c.End})
		}
		r.Types = append(r.Types, ty)
		return end
	}
	return k
}

// invClauses reads "invariant expr" clauses, each ending at a line end.
func invClauses(toks []lex.Tok, k int) ([]lex.Clause, int) {
	var out []lex.Clause
	for toks[k].Is("invariant") {
		c := lex.Clause{Word: "invariant", WordOff: toks[k].Off}
		k++
		c.Off = toks[k].Off
		depth := 0
		last := k
		for ; k < len(toks); k++ {
			t := toks[k]
			if depth == 0 && (t.Tok == token.SEMICOLON || t.Tok == token.EOF) {
				break
			}
			switch t.Tok {
			case token.LPAREN, token.LBRACK, token.LBRACE:
				depth++
			case token.RPAREN, token.RBRACK, token.RBRACE:
				depth--
			}
			last = k
		}
		c.End = toks[last].End
		out = append(out, c)
		if toks[k].AutoSemi() && toks[k+1].Is("invariant") {
			k++
			continue
		}
		break
	}
	return out, k
}

// apply replaces each edited range with spaces (keeping newlines and tabs), or inserts text.
func apply(src []byte, edits []edit) []byte {
	out := append([]byte(nil), src...)
	// Blank first, in place; then insert, from the end.
	var inserts []edit
	for _, e := range edits {
		if e.text != "" {
			inserts = append(inserts, e)
			continue
		}
		for k := e.off; k < e.end && k < len(out); k++ {
			if out[k] != '\n' && out[k] != '\r' && out[k] != '\t' {
				out[k] = ' '
			}
		}
	}
	for n := len(inserts) - 1; n >= 0; n-- {
		e := inserts[n]
		out = append(out[:e.off], append([]byte(e.text), out[e.off:]...)...)
	}
	return out
}

// limits finds "limit" on struct fields between toks[from] and toks[to], reports it as deferred,
// and blanks it to the end of its line or the field's tag. A field named limit, "limit int", is
// left alone: there "limit" starts the line and is not followed by a number.
func limits(toks []lex.Tok, from, to int, r *Result, edits *[]edit) {
	inStruct := 0
	for k := from; k < to && k < len(toks); k++ {
		t := toks[k]
		if t.Tok == token.STRUCT && toks[k+1].Tok == token.LBRACE {
			inStruct++
		}
		if inStruct == 0 || !t.Is("limit") {
			continue
		}
		prev, next := toks[k-1], toks[k+1]
		fieldStart := prev.Tok == token.SEMICOLON || prev.Tok == token.LBRACE || prev.Tok == token.COMMA
		if fieldStart && next.Tok != token.INT {
			continue
		}
		r.Deferred = append(r.Deferred, Deferred{What: "limit on a field", Section: "§5.5.1", Off: t.Off})
		end := k
		for end+1 < len(toks) && toks[end+1].Tok != token.SEMICOLON && toks[end+1].Tok != token.STRING && toks[end+1].Tok != token.RBRACE {
			end++
		}
		*edits = append(*edits, edit{off: t.Off, end: toks[end].End})
	}
}
