// Package contract checks contracts when the code runs: what a function requires, what it
// ensures, and the invariant of a type (PRIMER.md, section 3). A failed check panics with a
// *Failure. vulcan-lint checks where and how the calls are written.
package contract

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// Failure is the value of the panic of a failed contract.
type Failure struct {
	Kind string // requires, ensures, or invariant
	Func string // the function whose contract failed
	At   string // the file and line of the clause
	Type string // the type of an invariant
}

func (f *Failure) Error() string {
	if f.Kind == "invariant" {
		return fmt.Sprintf("contract failure: the invariant of %s is false after %s (%s)", f.Type, f.Func, f.At)
	}
	return fmt.Sprintf("contract failure: %s %s is false (%s)", f.Func, f.Kind, f.At)
}

// Requires checks a precondition. It is among the first statements of a function:
//
//	contract.Requires(amount > 0)
func Requires(cond bool) {
	if !cond {
		panic(&Failure{Kind: "requires", Func: caller(2), At: line(2)})
	}
}

// Ensures checks a postcondition when the function returns. It is deferred among the first
// statements of a function, and reads the named results:
//
//	defer contract.Ensures(func() bool { return err != nil || p.Amount == amount })
//
// When the function panics, Ensures checks nothing, and the panic goes on.
func Ensures(cond func() bool) {
	if r := recover(); r != nil {
		panic(r)
	}
	if !cond() {
		fn := runtime.FuncForPC(reflect.ValueOf(cond).Pointer())
		at := line(2)
		if fn != nil {
			file, n := fn.FileLine(fn.Entry())
			at = fmt.Sprintf("%s:%d", short(file), n)
		}
		panic(&Failure{Kind: "ensures", Func: caller(2), At: at})
	}
}

// Old returns its argument. It keeps the value at entry of a variable that a postcondition
// compares with, and holds no reference:
//
//	before := contract.Old(a.Balance)
func Old[T any](v T) T { return v }

// Checked is a type with an invariant.
type Checked interface{ Invariant() bool }

// Keep checks the invariant of the receiver when a method returns. It is deferred among the
// first statements of each method with a pointer receiver:
//
//	defer contract.Keep(a)
func Keep(v Checked) {
	if r := recover(); r != nil {
		panic(r)
	}
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Pointer && rv.IsNil() {
		return
	}
	if !v.Invariant() {
		panic(&Failure{Kind: "invariant", Func: caller(2), At: line(2), Type: typeName(v)})
	}
}

// Valid checks the invariant of a value that the code builds, and returns the value:
//
//	s := contract.Valid(shop.Stock{OnHand: n, Reserved: r})
func Valid[T Checked](v T) T {
	if !v.Invariant() {
		panic(&Failure{Kind: "invariant", Func: "the value built in " + caller(2), At: line(2), Type: typeName(v)})
	}
	return v
}

func typeName(v any) string { return strings.TrimPrefix(fmt.Sprintf("%T", v), "*") }

func caller(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return "?"
	}
	name := runtime.FuncForPC(pc).Name()
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return name
}

func line(skip int) string {
	_, file, n, ok := runtime.Caller(skip)
	if !ok {
		return "?"
	}
	return fmt.Sprintf("%s:%d", short(file), n)
}

// short keeps the last three elements of a path, such as api/billing/billing.go.
func short(file string) string {
	parts := strings.Split(file, "/")
	if len(parts) > 3 {
		parts = parts[len(parts)-3:]
	}
	return strings.Join(parts, "/")
}
