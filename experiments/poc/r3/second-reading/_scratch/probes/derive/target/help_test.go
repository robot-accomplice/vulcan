package target

import (
	"math/rand"
	"reflect"
)

func small(v []reflect.Value, r *rand.Rand) {
	for i := range v {
		v[i] = reflect.ValueOf(r.Intn(21) - 10)
	}
}
