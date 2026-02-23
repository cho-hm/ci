package env

var registry = map[string]Environment{}

func Register(envType string, e Environment) {
	registry[envType] = e
}

func Of(envType string) Environment {
	e, ok := registry[envType]
	if !ok {
		panic("unsupported env type: " + envType)
	}
	return e
}
