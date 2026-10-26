// Package decl parses declaration files: system.vulcan, component.vulcan, and library.vulcan
// (spec Appendix B.2).
package decl

import "go/ast"

// Pos is a byte offset in the file being parsed.
type Pos = int

// Import is one import line.
type Import struct {
	Name string // the explicit package name, or ""
	Path string
	Pos  Pos
}

// System is the content of system.vulcan.
type System struct {
	Name       string
	Pos        Pos
	Imports    []Import
	Persons    []*Node
	Externals  []*Node
	Containers []*Container
	Wires      []*Wire // wires outside any container (between containers)
	Config     *Settings
	Uses       []Pos
}

// Node is a person or an external system.
type Node struct {
	Name, Desc string
	Pos        Pos
}

// Container is one container section.
type Container struct {
	Name   string
	Pos    Pos
	Leaves []*LeafEntry // every leaf, also those inside groups
	Groups []*Group
	Wires  []*Wire
}

// Group is a label for diagrams.
type Group struct {
	Name string
	Pos  Pos
}

// LeafEntry is one leaf in a container list.
type LeafEntry struct {
	Name     string
	Pos      Pos
	Group    string
	Settings *Settings
}

// Endpoint is [Container "/"] Leaf "." Name.
type Endpoint struct {
	Container string
	Leaf      string
	Name      string
	Pos       Pos // of the first identifier
	NamePos   Pos
}

// Wire is "needer: supplier".
type Wire struct {
	Need     Endpoint
	Supply   Endpoint
	Settings *Settings
	Via      string
	ViaPos   Pos
	Pos      Pos
}

// Settings is "{ key: value, ... }".
type Settings struct {
	Pos   Pos
	Items []*Setting
}

// Setting is one "key: value".
type Setting struct {
	Key    string
	KeyPos Pos
	Value  *Value
}

// ValueKind tells the form of a value.
type ValueKind int

const (
	ExprValue ValueKind = iota
	FlagValue
	ListValue
	NamedSettingsValue
)

// Value is the value of a setting.
type Value struct {
	Kind     ValueKind
	Pos, End Pos
	Text     string   // the source text, for ExprValue
	Expr     ast.Expr // for ExprValue
	Flag     *Flag
	List     []*Value
	Name     string // for NamedSettingsValue
	Settings *Settings
}

// Flag is flag("name", default: v, max: n).
type Flag struct {
	Name    string
	Pos     Pos
	Default *Value
	Max     *Value
}

// Component is the content of a component.vulcan file.
type Component struct {
	Name     string
	Pos      Pos
	Imports  []Import
	Ports    []*Port
	Gateways []*Node // "gateway for X": Name is X
	Entries  []*Node // "entry for X"
	// Deferred constructs, kept for K0009.
	Deferred []Deferred
}

// Deferred is a construct that the release does not support.
type Deferred struct {
	What    string
	Section string
	Pos     Pos
}

// Port is "needs port Name { methods }".
type Port struct {
	Name    string
	Pos     Pos
	Remote  bool
	Methods []*PortMethod
}

// PortMethod is one method of a port.
type PortMethod struct {
	Name      string
	Pos       Pos
	SigOff    int // byte offsets of the signature text after the name
	SigEnd    int
	Contracts []Contract
}

// Contract is one requires or ensures clause.
type Contract struct {
	Word     string
	WordPos  Pos
	Off, End int
}

// Library is the content of a library.vulcan file.
type Library struct {
	Name    string
	Pos     Pos
	Imports []Import
}
