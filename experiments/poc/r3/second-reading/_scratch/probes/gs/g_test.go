package gs

import (
	"fmt"
	"testing"
	"time"
)

func TestSim(t *testing.T) {
	m := map[int]int{1: 1, 2: 2, 3: 3}
	for k := range m {
		fmt.Println(k)
	}
	go func() { time.Sleep(time.Hour); fmt.Println("late") }()
	time.Sleep(2 * time.Hour)
	fmt.Println(time.Now())
}
