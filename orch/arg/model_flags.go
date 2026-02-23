package arg

type Flags struct {
	parse       bool // only parse
	check       bool // parse and check
	publish     bool // parse, check and publish
	build       bool // parse, check and build
	envType     string
	initialized bool
}

func (f Flags) Parse() bool {
	return f.parse
}
func (f Flags) Check() bool {
	return f.check
}
func (f Flags) Publish() bool {
	return f.publish
}
func (f Flags) Build() bool {
	return f.build
}
func (f Flags) EnvType() string {
	return f.envType
}
