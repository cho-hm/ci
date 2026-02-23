package arg

func buildSingletonFlags(parse, check, publish, build bool, envType string) Flags {
	ret := Flags{
		build:   false,
		publish: false,
		check:   true,
		parse:   true,
		envType: envType,
	}
	tasks = 2
	if !parse && !check && !publish && !build {
		panic("Need some options, but nothing.\nSee -h (help)")
	} else if build || publish {
		ret.build = build
		ret.publish = publish
		if ret.build {
			tasks++
		}
		if ret.publish {
			tasks++
		}
	} else if parse {
		ret.check = false
		tasks = 1
	}
	return ret
}
