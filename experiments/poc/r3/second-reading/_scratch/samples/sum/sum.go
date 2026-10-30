package sum

// Shape is a sum type: sealed by an unexported method.
//
//sumtype:decl
type Shape interface{ isShape() }

type Circle struct{ R int }
type Square struct{ S int }

func (Circle) isShape() {}
func (Square) isShape() {}

func Area(s Shape) int {
	switch v := s.(type) {
	case Circle:
		return v.R * v.R * 3
	}
	return 0
}

func Nil() int { var s Shape; return Area(s) }

type Color int

const (
	Red Color = iota
	Blue
)

func Name(c Color) string {
	switch c {
	case Red:
		return "red"
	}
	return ""
}
