package lin

//vl:linear
type Conn struct{ fd int }

func Open() *Conn { return &Conn{1} }

//vl:consumes
func (c *Conn) Close() {}

func (c *Conn) Send(b byte) {}

func good() {
	c := Open()
	c.Send(1)
	c.Close()
}

func deferred() {
	c := Open()
	defer c.Close()
	c.Send(1)
}

func moved() *Conn {
	c := Open()
	return c
}

func forgot(x bool) {
	c := Open()
	if x {
		c.Close()
		return
	}
	c.Send(2)
}

func twice() {
	c := Open()
	c.Close()
	c.Close()
}

func afterClose() {
	c := Open()
	c.Close()
	c.Send(3)
}

func captured() {
	c := Open()
	f := func() { c.Close() }
	f()
	c.Close()
}
