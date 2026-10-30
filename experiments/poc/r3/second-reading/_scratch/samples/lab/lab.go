package lab

import (
	"fmt"
	"slices"
	"strings"
)

//vl:label
type Email string

type User struct {
	ID   int
	Mail Email
}

// Hash is label-polymorphic with an ordinary type parameter.
func Trim[S ~string](s S) S { return S(strings.TrimSpace(string(s))) }

func ok(a, b Email) Email { return a + b }

func viaWrap(e Email) Email { return Trim(e) }

func leaks(u User, e Email, es []Email) {
	fmt.Println(e)
	fmt.Println(u)
	var x any = e
	_ = x
	_ = string(e)
	_ = e[0]
	for range e {
	}
	if e == "root@x" {
		fmt.Println("implicit")
	}
	_ = len(e)
	_ = slices.Clone(es)
}
