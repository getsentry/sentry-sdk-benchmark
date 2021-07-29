package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var usage = `
Usage:	%[1]s PLATFORM [PLATFORM ...]

Examples:
%[1]s platform/python/django
%[1]s platform/javascript/express
`

func main() {
	flag.Parse()
	if len(flag.Args()) < 1 {
		fmt.Fprintf(os.Stderr, strings.TrimSpace(usage), os.Args[0])
		fmt.Fprintln(os.Stderr)
		os.Exit(2)
	}
	for _, platform := range flag.Args() {
		bench(platform)
	}
}
