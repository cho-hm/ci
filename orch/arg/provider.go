package arg

// Flag - Getter for readonly singleton object type of flags.
func Flag() Flags {
	if !singletonFlags.initialized {
		parse()
	}
	return singletonFlags
}
func Task() int {
	return tasks
}
