// Package arch is the manifest of the architecture: every package of the module, with its kind
// (PRIMER.md, section 1). vulcan-lint reads it. The build type-checks it, so a name that is not
// declared is a compile error. This file is the schema of the manifest, the same in every
// project; arch.go is the manifest.
package arch

// Person is someone who uses the system.
type Person string

// External is a system that the system uses.
type External string

// Party is a Person or an External.
type Party interface{ party() }

func (Person) party()   {}
func (External) party() {}

// Part is one package of the module, with its kind.
type Part struct {
	dir   string
	kind  string
	party Party
}

// Leaf is a leaf: it uses other leaves only through its ports.
func Leaf(dir string) Part { return Part{dir, "leaf", nil} }

// Gateway is the one leaf that talks to an external system.
func Gateway(dir string, of External) Part { return Part{dir, "gateway", of} }

// Entry is a leaf that a person or an external system calls: it serves.
func Entry(dir string, of Party) Part { return Part{dir, "entry", of} }

// Library is shared code in lib/: no ports, no state that changes.
func Library(dir string) Part { return Part{dir, "library", nil} }

// Main is a main package: it wires the leaves.
func Main(dir string) Part { return Part{dir, "main", nil} }

// System is the manifest.
type System struct {
	Parts []Part

	// Strict turns on strict mode (PRIMER.md, section 2).
	Strict bool
	// StrictExempt lists leaves that strict mode skips.
	StrictExempt []string
	// StrictInterfaces lists interface types that a port method may carry, such as "io.Reader".
	StrictInterfaces []string
	// StrictShared lists types that a library variable may have although they hold a reference.
	StrictShared []string
	// Mutators lists more functions that change process-wide state, by types.Func.FullName.
	Mutators []string
}
