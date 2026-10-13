package picking

import (
	"errors"
	"strings"
	"testing"

	"shop/shop"
)

type recorder struct{ got []shop.OrderShipped }

func (r *recorder) Publish(ev shop.OrderShipped) { r.got = append(r.got, ev) }

// The note limit protects the event size; an oversized note must not be published.
func TestShipEnforcesNoteLimit(t *testing.T) {
	r := &recorder{}
	p := New(r)
	if err := p.Ship("o-1", strings.Repeat("x", shop.MaxNoteBytes)); err != nil {
		t.Fatalf("note at the limit: %v", err)
	}
	if err := p.Ship("o-2", strings.Repeat("x", shop.MaxNoteBytes+1)); !errors.Is(err, ErrNoteTooBig) {
		t.Fatalf("note over the limit: got %v", err)
	}
	if len(r.got) != 1 || r.got[0].Order != "o-1" {
		t.Fatalf("published %+v", r.got)
	}
}
