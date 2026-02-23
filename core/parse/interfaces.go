package parse

type ChannelProvider[T any] interface {
	Get() chan T
}

type Context interface {
	init()
}

type ContextProvider[T Context] interface {
	Get() T
}
