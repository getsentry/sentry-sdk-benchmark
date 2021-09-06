package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var usage = `
Usage:	%[1]s [run] PLATFORM [PLATFORM ...]

Benchmark one or more platforms.

Examples:
%[1]s platform/python/django
%[1]s run platform/javascript/express

Usage:	%[1]s report RESULT [RESULT ...]

Print an HTML report summarizing the results of one or more benchmark runs.
Reports are automatically created after a successful benchmark run.
This subcommand allows re-generating a report from benchmark result data on demand.

Examples:
%[1]s report result/python/django/20210818-082527-tbnfsga
%[1]s report result/python/django/20210818-*
`

func printUsage() {
	fmt.Fprintf(os.Stderr, strings.TrimSpace(usage), filepath.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lmsgprefix)
	log.SetPrefix("[sentry-sdk-benchmark] ")

	defer func() {
		if err := recover(); err != nil {
			_, file, line, ok := runtime.Caller(2)
			if !ok {
				panic(err)
			}
			log.Fatalf("Failure: %s:%d: %s", file, line, err)
		}
	}()

	flag.Parse()
	if len(flag.Args()) < 1 {
		printUsage()
		os.Exit(2)
	}

	switch args := flag.Args(); args[0] {
	case "report":
		args = args[1:]
		if len(args) == 0 {
			printUsage()
			os.Exit(2)
		}
		Report(args)
	case "run":
		args = args[1:]
		fallthrough
	default:
		if len(args) == 0 {
			printUsage()
			os.Exit(2)
		}
		for _, platform := range args {
			Benchmark(BenchmarkConfigFromPlatform(platform))
		}
	}
}
