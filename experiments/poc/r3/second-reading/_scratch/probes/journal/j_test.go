// T18 on plain Go: a generated wrapper of a port records each effect in a journal, and a handler
// runs again from its start after a crash. The handler text stays plain Go.
package journal

import (
	"fmt"
	"testing"
)

type Store interface {
	Charge(amount int) (string, error)
}

type realStore struct{ charges int }

func (r *realStore) Charge(a int) (string, error) {
	r.charges++
	return fmt.Sprintf("p%d", r.charges), nil
}

// Journal stands for a table that is written in one transaction per step.
type Journal struct{ steps map[string]string }

// journaledStore is what a generator writes from the port declaration.
type journaledStore struct {
	inner       Store
	j           *Journal
	id          string
	n           int
	crashBefore int // crash after the effect of this step, before its record
	crashAfter  int // crash after the record of this step
}

func (s *journaledStore) Charge(a int) (string, error) {
	s.n++
	key := fmt.Sprintf("%s/%d", s.id, s.n)
	if v, ok := s.j.steps[key]; ok {
		return v, nil
	}
	p, err := s.inner.Charge(a)
	if s.n == s.crashBefore {
		panic("crash")
	}
	s.j.steps[key] = p
	if s.n == s.crashAfter {
		panic("crash")
	}
	return p, err
}

// handler is the plain Go that the agent writes.
func handler(st Store) ([]string, error) {
	var out []string
	for _, a := range []int{10, 20, 30} {
		p, err := st.Charge(a)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func run(st Store) (out []string, crashed bool) {
	defer func() {
		if r := recover(); r != nil {
			crashed = true
		}
	}()
	out, _ = handler(st)
	return
}

func TestCrashAfterRecord(t *testing.T) {
	r := &realStore{}
	j := &Journal{map[string]string{}}
	if _, crashed := run(&journaledStore{inner: r, j: j, id: "m1", crashAfter: 2}); !crashed {
		t.Fatal("no crash")
	}
	out, crashed := run(&journaledStore{inner: r, j: j, id: "m1"})
	t.Log(out, "charges", r.charges)
	if crashed || r.charges != 3 || out[2] != "p3" {
		t.Fatal(out, r.charges)
	}
}

func TestCrashBeforeRecord(t *testing.T) {
	r := &realStore{}
	j := &Journal{map[string]string{}}
	run(&journaledStore{inner: r, j: j, id: "m1", crashBefore: 2})
	out, _ := run(&journaledStore{inner: r, j: j, id: "m1"})
	t.Log(out, "charges", r.charges, "(the window between an effect and its record needs an idempotency key)")
}
