package arg

import (
	"flag"
	"log"
	"os"
	"strings"
)

/*
parse
Check and set execution flags to singleton object type flags.
*/
func parse() {
	if singletonFlags.initialized {
		return
	}
	var parse, check, publish, build bool
	var envType string
	log.Println("Start to parse flags...")
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.BoolVar(&parse, "parse", false, "If want only parse")
	fs.BoolVar(&check, "check", false, "If want parse and check")
	fs.BoolVar(&publish, "publish", false, "If want parse, check and publish")
	fs.BoolVar(&build, "build", false, "If want parse, check and build")
	fs.StringVar(&envType, "env", "gradle", "Build environment type: gradle, node")
	fs.Parse(filterTestArgs(os.Args[1:]))
	singletonFlags = buildSingletonFlags(parse, check, publish, build, envType)
	singletonFlags.initialized = true
	log.Printf("%c => Done!\n", '\u2714')
}

func filterTestArgs(args []string) []string {
	var filtered []string
	for _, a := range args {
		if !strings.HasPrefix(a, "-test.") {
			filtered = append(filtered, a)
		}
	}
	return filtered
}
