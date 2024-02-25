package httpserver

type cache struct {
	sema chan struct{}
	data []byte
}

func mkCache() *cache {
	return &cache{
		sema: make(chan struct{}, 1),
	}
}

func (c *cache) lock() *cache {
	c.sema <- struct{}{}
	return c
}

func (c *cache) unlock() {
	<-c.sema
}

func (c *cache) write(data []byte) {
	defer c.lock().unlock()
	c.data = data
}

func (c *cache) read() []byte {
	defer c.lock().unlock()
	return c.data
}
