package parse

import "sync"

type StateChannelProvider struct {
	once    sync.Once
	channel chan int
}

func (p *StateChannelProvider) Get() chan int {
	p.once.Do(func() {
		p.channel = make(chan int, 1)
	})
	return p.channel
}

type DefaultContextProvider[T Context] struct {
	context         T
	once            sync.Once
	newInstanceFunc func() T
}

func (b *DefaultContextProvider[T]) Get() T {
	if b.newInstanceFunc == nil {
		panic("Require property: newInstanceFunc, but actual nil")
	}
	b.once.Do(func() {
		b.context = b.newInstanceFunc()
		b.context.init()
	})
	return b.context
}
