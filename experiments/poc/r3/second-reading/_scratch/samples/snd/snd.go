package snd

import "sync"

type Buf struct{ b []byte }

//vl:send
type Counter struct {
	mu sync.Mutex
	n  int
}

func race() int {
	n := 0
	go func() { n++ }()
	return n
}

func sendPtr(c chan *Buf, b *Buf) {
	c <- b
	b.b = nil
}

func goArg(b *Buf) { go use(b) }

func use(*Buf) {}

func captureSlice(xs []int) {
	go func() { xs[0] = 1 }()
}

func fine(c *Counter, ch chan int) {
	go func() {
		c.mu.Lock()
		c.n++
		c.mu.Unlock()
		ch <- 1
	}()
}
