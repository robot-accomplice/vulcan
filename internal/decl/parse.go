package decl

import (
	"fmt"
	"go/parser"
	"go/token"
	"strconv"

	"github.com/robot-accomplice/vulcan/internal/lex"
)

// Problem is a parse diagnostic at a byte offset.
type Problem struct {
	Off  int
	Code string
	Msg  string
}

type p struct {
	src   []byte
	toks  []lex.Tok
	i     int
	probs []Problem
}

type bail struct{}

func newParser(src []byte) *p {
	toks, errs := lex.Scan(src)
	ps := &p{src: src, toks: toks}
	for _, e := range errs {
		ps.probs = append(ps.probs, Problem{Off: e.Off, Code: "K0001", Msg: e.Msg})
	}
	return ps
}

func (ps *p) tok() lex.Tok  { return ps.toks[ps.i] }
func (ps *p) peek() lex.Tok { return ps.toks[min(ps.i+1, len(ps.toks)-1)] }
func (ps *p) next() lex.Tok {
	t := ps.toks[ps.i]
	if ps.i < len(ps.toks)-1 {
		ps.i++
	}
	return t
}

func (ps *p) fail(off int, format string, args ...any) {
	ps.probs = append(ps.probs, Problem{Off: off, Code: "K0001", Msg: fmt.Sprintf(format, args...)})
	panic(bail{})
}

func describe(t lex.Tok) string {
	switch {
	case t.Tok == token.EOF:
		return "end of file"
	case t.AutoSemi():
		return "newline"
	case t.Lit != "":
		return strconv.Quote(t.Lit)
	}
	return strconv.Quote(t.Tok.String())
}

func (ps *p) expect(tk token.Token) lex.Tok {
	t := ps.tok()
	if t.Tok != tk {
		ps.fail(t.Off, "expected %q, found %s", tk.String(), describe(t))
	}
	return ps.next()
}

func (ps *p) ident() lex.Tok {
	t := ps.tok()
	if t.Tok != token.IDENT {
		ps.fail(t.Off, "expected a name, found %s", describe(t))
	}
	return ps.next()
}

func (ps *p) word(w string) lex.Tok {
	t := ps.tok()
	if !t.Is(w) {
		ps.fail(t.Off, "expected %q, found %s", w, describe(t))
	}
	return ps.next()
}

// skipSemis skips semicolons.
func (ps *p) skipSemis() {
	for ps.tok().Tok == token.SEMICOLON {
		ps.next()
	}
}

// skipNewlines skips semicolons that the scanner inserted, inside brackets.
func (ps *p) skipNewlines() {
	for ps.tok().AutoSemi() {
		ps.next()
	}
}

// endItem ends an item: a semicolon, or the "}" that closes the section.
func (ps *p) endItem() {
	switch ps.tok().Tok {
	case token.SEMICOLON:
		ps.next()
	case token.RBRACE, token.EOF:
	default:
		ps.fail(ps.tok().Off, "expected a newline, found %s", describe(ps.tok()))
	}
}

func (ps *p) imports() []Import {
	var out []Import
	ps.skipSemis()
	for ps.tok().Tok == token.IMPORT {
		ps.next()
		if ps.tok().Tok == token.LPAREN {
			ps.next()
			ps.skipSemis()
			for ps.tok().Tok != token.RPAREN && ps.tok().Tok != token.EOF {
				out = append(out, ps.importSpec())
				ps.skipSemis()
			}
			ps.expect(token.RPAREN)
		} else {
			out = append(out, ps.importSpec())
		}
		ps.skipSemis()
	}
	return out
}

func (ps *p) importSpec() Import {
	var imp Import
	if ps.tok().Tok == token.IDENT || ps.tok().Tok == token.PERIOD {
		imp.Name = ps.next().Lit
		if imp.Name == "" {
			imp.Name = "."
		}
	}
	t := ps.expect(token.STRING)
	path, err := strconv.Unquote(t.Lit)
	if err != nil {
		ps.fail(t.Off, "bad import path %s", t.Lit)
	}
	imp.Path = path
	imp.Pos = t.Off
	return imp
}

func run(ps *p, f func()) (probs []Problem) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(bail); !ok {
				panic(r)
			}
		}
		probs = ps.probs
	}()
	f()
	return
}

// ParseSystem parses system.vulcan.
func ParseSystem(src []byte) (*System, []Problem) {
	ps := newParser(src)
	sys := &System{}
	probs := run(ps, func() {
		sys.Imports = ps.imports()
		ps.word("system")
		name := ps.ident()
		sys.Name, sys.Pos = name.Lit, name.Off
		ps.expect(token.LBRACE)
		for {
			ps.skipSemis()
			t := ps.tok()
			if t.Tok == token.RBRACE {
				ps.next()
				break
			}
			if t.Tok == token.EOF {
				ps.fail(t.Off, "missing \"}\" at the end of system %s", sys.Name)
			}
			switch {
			case t.Is("container") && ps.peek().Tok == token.IDENT:
				sys.Containers = append(sys.Containers, ps.container())
			case t.Is("person") && ps.peek().Tok == token.IDENT:
				sys.Persons = append(sys.Persons, ps.node())
			case t.Is("external") && ps.peek().Tok == token.IDENT:
				sys.Externals = append(sys.Externals, ps.node())
			case t.Is("config") && ps.peek().Tok == token.LBRACE:
				ps.next()
				if sys.Config != nil {
					ps.probs = append(ps.probs, Problem{Off: t.Off, Code: "K0001", Msg: "a second config section"})
				}
				sys.Config = ps.settings()
			case t.Is("uses") && ps.peek().Tok == token.IDENT:
				sys.Uses = append(sys.Uses, t.Off)
				ps.next()
				ps.nodeRef()
				ps.nodeRef()
				ps.expect(token.STRING)
			case t.Tok == token.IDENT:
				sys.Wires = append(sys.Wires, ps.wire())
			default:
				ps.fail(t.Off, "expected a container, person, external, config, uses, or wire, found %s", describe(t))
			}
			ps.endItem()
		}
		ps.skipSemis()
		if ps.tok().Tok != token.EOF {
			ps.fail(ps.tok().Off, "unexpected %s after the system", describe(ps.tok()))
		}
	})
	return sys, probs
}

func (ps *p) nodeRef() {
	ps.ident()
	if ps.tok().Tok == token.QUO {
		ps.next()
		ps.ident()
	}
}

func (ps *p) node() *Node {
	ps.next()
	name := ps.ident()
	desc := ps.expect(token.STRING)
	d, _ := strconv.Unquote(desc.Lit)
	return &Node{Name: name.Lit, Pos: name.Off, Desc: d}
}

func (ps *p) container() *Container {
	ps.next()
	name := ps.ident()
	c := &Container{Name: name.Lit, Pos: name.Off}
	ps.expect(token.LBRACE)
	ps.containerItems(c, "")
	return c
}

// containerItems reads items up to and including the closing "}".
func (ps *p) containerItems(c *Container, group string) {
	for {
		ps.skipSemis()
		t := ps.tok()
		if t.Tok == token.RBRACE {
			ps.next()
			return
		}
		if t.Tok == token.EOF {
			ps.fail(t.Off, "missing \"}\"")
		}
		switch {
		case t.Is("group") && ps.peek().Tok == token.IDENT:
			ps.next()
			name := ps.ident()
			c.Groups = append(c.Groups, &Group{Name: name.Lit, Pos: name.Off})
			ps.expect(token.LBRACE)
			ps.containerItems(c, name.Lit)
		case t.Tok == token.IDENT && (ps.peek().Tok == token.PERIOD || ps.peek().Tok == token.QUO):
			if group != "" {
				ps.fail(t.Off, "a wire inside group %s; wires go in the container section", group)
			}
			c.Wires = append(c.Wires, ps.wire())
		case t.Tok == token.IDENT:
			for {
				name := ps.ident()
				le := &LeafEntry{Name: name.Lit, Pos: name.Off, Group: group}
				if ps.tok().Tok == token.LBRACE {
					le.Settings = ps.settings()
				}
				c.Leaves = append(c.Leaves, le)
				if ps.tok().Tok != token.COMMA {
					break
				}
				ps.next()
				ps.skipNewlines()
			}
		default:
			ps.fail(t.Off, "expected a leaf, a group, or a wire, found %s", describe(t))
		}
		ps.endItem()
	}
}

func (ps *p) endpoint() Endpoint {
	first := ps.ident()
	e := Endpoint{Pos: first.Off}
	leaf := first
	if ps.tok().Tok == token.QUO {
		ps.next()
		e.Container = first.Lit
		leaf = ps.ident()
	}
	e.Leaf = leaf.Lit
	ps.expect(token.PERIOD)
	n := ps.ident()
	e.Name, e.NamePos = n.Lit, n.Off
	return e
}

func (ps *p) wire() *Wire {
	w := &Wire{Pos: ps.tok().Off}
	w.Need = ps.endpoint()
	ps.expect(token.COLON)
	w.Supply = ps.endpoint()
	if ps.tok().Tok == token.LBRACE {
		w.Settings = ps.settings()
	}
	if ps.tok().Is("via") {
		ps.next()
		v := ps.ident()
		w.Via, w.ViaPos = v.Lit, v.Off
		if ps.tok().Tok == token.LBRACE {
			ps.settings()
		}
	}
	return w
}

func (ps *p) settings() *Settings {
	open := ps.expect(token.LBRACE)
	s := &Settings{Pos: open.Off}
	for {
		ps.skipNewlines()
		if ps.tok().Tok == token.RBRACE {
			ps.next()
			return s
		}
		key := ps.ident()
		ps.expect(token.COLON)
		st := &Setting{Key: key.Lit, KeyPos: key.Off, Value: ps.value()}
		s.Items = append(s.Items, st)
		ps.skipNewlines()
		if ps.tok().Tok == token.COMMA {
			ps.next()
			continue
		}
		ps.skipNewlines()
		if ps.tok().Tok != token.RBRACE {
			ps.fail(ps.tok().Off, "expected \",\" or \"}\" in settings, found %s", describe(ps.tok()))
		}
	}
}

func (ps *p) value() *Value {
	ps.skipNewlines()
	t := ps.tok()
	switch {
	case t.Is("flag") && ps.peek().Tok == token.LPAREN:
		return ps.flag()
	case t.Tok == token.LBRACK:
		ps.next()
		v := &Value{Kind: ListValue, Pos: t.Off}
		for {
			ps.skipNewlines()
			if ps.tok().Tok == token.RBRACK {
				v.End = ps.next().End
				return v
			}
			v.List = append(v.List, ps.value())
			ps.skipNewlines()
			if ps.tok().Tok == token.COMMA {
				ps.next()
			} else if ps.tok().Tok != token.RBRACK {
				ps.fail(ps.tok().Off, "expected \",\" or \"]\" in a list, found %s", describe(ps.tok()))
			}
		}
	case t.Tok == token.IDENT && ps.peek().Tok == token.LBRACE:
		ps.next()
		v := &Value{Kind: NamedSettingsValue, Pos: t.Off, Name: t.Lit}
		v.Settings = ps.settings()
		v.End = ps.toks[ps.i-1].End
		return v
	}
	return ps.expr()
}

// expr reads a Go expression up to a "," or a closing bracket at depth 0.
func (ps *p) expr() *Value {
	start := ps.i
	depth := 0
	for {
		t := ps.tok()
		if t.Tok == token.EOF {
			break
		}
		if depth == 0 && (t.Tok == token.COMMA || t.Tok == token.RBRACE || t.Tok == token.RBRACK || t.Tok == token.RPAREN || t.Tok == token.SEMICOLON) {
			break
		}
		switch t.Tok {
		case token.LPAREN, token.LBRACK, token.LBRACE:
			depth++
		case token.RPAREN, token.RBRACK, token.RBRACE:
			depth--
		}
		ps.next()
	}
	if ps.i == start {
		ps.fail(ps.tok().Off, "expected a value, found %s", describe(ps.tok()))
	}
	off, end := ps.toks[start].Off, ps.toks[ps.i-1].End
	text := string(ps.src[off:end])
	e, err := parser.ParseExpr(text)
	if err != nil {
		ps.fail(off, "bad value %q: %v", text, err)
	}
	return &Value{Kind: ExprValue, Pos: off, End: end, Text: text, Expr: e}
}

func (ps *p) flag() *Value {
	start := ps.next()
	ps.expect(token.LPAREN)
	name := ps.expect(token.STRING)
	n, err := strconv.Unquote(name.Lit)
	if err != nil || n == "" {
		ps.fail(name.Off, "a flag needs a name")
	}
	f := &Flag{Name: n, Pos: name.Off}
	for ps.tok().Tok == token.COMMA {
		ps.next()
		ps.skipNewlines()
		opt := ps.tok()
		if opt.Tok == token.DEFAULT {
			opt.Lit = "default"
			ps.next()
		} else {
			opt = ps.ident()
		}
		ps.expect(token.COLON)
		v := ps.expr()
		switch opt.Lit {
		case "default":
			f.Default = v
		case "max":
			f.Max = v
		default:
			ps.fail(opt.Off, "unknown flag option %q; use default or max", opt.Lit)
		}
	}
	ps.skipNewlines()
	end := ps.expect(token.RPAREN)
	return &Value{Kind: FlagValue, Pos: start.Off, End: end.End, Flag: f}
}

// ParseComponent parses a component.vulcan file.
func ParseComponent(src []byte) (*Component, []Problem) {
	ps := newParser(src)
	c := &Component{}
	probs := run(ps, func() {
		c.Imports = ps.imports()
		ps.word("component")
		name := ps.ident()
		c.Name, c.Pos = name.Lit, name.Off
		ps.expect(token.LBRACE)
		for {
			ps.skipSemis()
			t := ps.tok()
			if t.Tok == token.RBRACE {
				ps.next()
				break
			}
			switch {
			case t.Is("needs"):
				ps.next()
				port := &Port{}
				if ps.tok().Is("remote") {
					port.Remote = true
					c.Deferred = append(c.Deferred, Deferred{What: "needs remote port", Section: "§5.5.3", Pos: ps.tok().Off})
					ps.next()
				}
				ps.word("port")
				n := ps.ident()
				port.Name, port.Pos = n.Lit, n.Off
				ps.expect(token.LBRACE)
				ps.portMethods(port, c)
				c.Ports = append(c.Ports, port)
			case (t.Is("gateway") || t.Is("entry")) && ps.peek().Tok == token.FOR:
				ps.next()
				ps.next()
				n := ps.ident()
				node := &Node{Name: n.Lit, Pos: n.Off}
				if t.Lit == "gateway" {
					c.Gateways = append(c.Gateways, node)
				} else {
					c.Entries = append(c.Entries, node)
				}
			case t.Is("publishes") || t.Is("subscribes"):
				c.Deferred = append(c.Deferred, Deferred{What: t.Lit + " (topics)", Section: "§5.5", Pos: t.Off})
				ps.next()
				ps.ident()
				ps.word("topic")
				lb := ps.expect(token.LBRACK)
				_ = lb
				ps.i = lex.Match(ps.toks, ps.i-1) + 1
			case t.Tok == token.EOF:
				ps.fail(t.Off, "missing \"}\" at the end of component %s", c.Name)
			default:
				ps.fail(t.Off, "expected needs port, gateway for, entry for, publishes, or subscribes, found %s", describe(t))
			}
			ps.endItem()
		}
		ps.skipSemis()
		if ps.tok().Tok != token.EOF {
			ps.fail(ps.tok().Off, "unexpected %s after the component", describe(ps.tok()))
		}
	})
	return c, probs
}

var portWords = map[string]bool{"requires": true, "ensures": true, "timeout": true}

func (ps *p) portMethods(port *Port, c *Component) {
	for {
		ps.skipSemis()
		t := ps.tok()
		if t.Tok == token.RBRACE {
			ps.next()
			return
		}
		if t.Tok != token.FUNC {
			ps.fail(t.Off, "expected func in port %s, found %s", port.Name, describe(t))
		}
		ps.next()
		n := ps.ident()
		m := &PortMethod{Name: n.Lit, Pos: n.Off}
		if ps.tok().Tok == token.LBRACK {
			ps.fail(ps.tok().Off, "a port method cannot have type parameters")
		}
		if ps.tok().Tok != token.LPAREN {
			ps.fail(ps.tok().Off, "expected \"(\" after port method %s, found %s", n.Lit, describe(ps.tok()))
		}
		end := lex.SigEnd(ps.toks, ps.i, portWords)
		m.SigOff = ps.tok().Off
		m.SigEnd = ps.toks[end-1].End
		ps.i = end
		clauses, k := lex.Clauses(ps.toks, ps.i, portWords)
		for _, cl := range clauses {
			if cl.Word == "timeout" {
				c.Deferred = append(c.Deferred, Deferred{What: "timeout on a port method", Section: "§5.5.3", Pos: cl.WordOff})
				continue
			}
			m.Contracts = append(m.Contracts, Contract{Word: cl.Word, WordPos: cl.WordOff, Off: cl.Off, End: cl.End})
		}
		ps.i = k
		port.Methods = append(port.Methods, m)
		ps.endItem()
	}
}

// ParseLibrary parses a library.vulcan file.
func ParseLibrary(src []byte) (*Library, []Problem) {
	ps := newParser(src)
	l := &Library{}
	probs := run(ps, func() {
		l.Imports = ps.imports()
		ps.word("library")
		name := ps.ident()
		l.Name, l.Pos = name.Lit, name.Off
		ps.expect(token.LBRACE)
		ps.skipSemis()
		if t := ps.tok(); t.Is("needs") || t.Is("publishes") || t.Is("subscribes") {
			ps.probs = append(ps.probs, Problem{Off: t.Off, Code: "K0201", Msg: "a library cannot declare a port or a topic (§5.4a rule 1)"})
			panic(bail{})
		}
		if ps.tok().Tok != token.RBRACE {
			ps.fail(ps.tok().Off, "a library declaration holds nothing; put code in .vulcan code files (K0201 if it is a port or a topic)")
		}
		ps.next()
		ps.skipSemis()
		if ps.tok().Tok != token.EOF {
			ps.fail(ps.tok().Off, "unexpected %s after the library", describe(ps.tok()))
		}
	})
	return l, probs
}
