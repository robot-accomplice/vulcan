package nz

//vl:nozero
type Email struct{ v string }

func New(s string) (Email, error) { return Email{v: s}, nil }

type User struct {
	M Email
	N int
}

func bad(m map[string]Email) {
	var e Email
	_ = e
	_ = new(Email)
	_ = User{N: 1}
	_ = make([]Email, 3)
	_ = m["x"]
	_ = zero[Email]()
	var arr [2]User
	_ = arr
}

func named() (e Email, err error) { return }

func zero[T any]() T {
	var z T
	return z
}

func good() User {
	e, _ := New("x")
	return User{M: e, N: 1}
}

// The Go idiom on an error path needs a zero, so the rule must take a pointer or a special case.
func errorPath(s string) (Email, error) {
	if s == "" {
		return Email{}, nil
	}
	return New(s)
}
