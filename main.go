package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
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

Usage:	%[1]s compare RESULT [RESULT ...]

Compares various runs using benchstat.

Examples:
%[1]s compare result/python/django/20210818-082527-tbnfsga result/platform/python/django/20210909-150838-bcvjada
%[1]s compare result/python/django/20210818-*
`

func printUsage() {
	fmt.Fprintf(os.Stderr, strings.TrimSpace(usage), filepath.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr)
}

// openBrowser controls whether to open a web browser to show HTML reports. The
// default behavior is to open a browser, unless running multiple benchmarks
// from a single command line execution.
var openBrowser = true

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	flag.BoolVar(&openBrowser, "browser", true, "open report in browser")

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
		if len(args) > 1 {
			openBrowser = false
		}
		Report(args)
	case "compare":
		args = args[1:]
		if len(args) == 0 {
			printUsage()
			os.Exit(2)
		}
		Compare(args)
	case "run":
		args = args[1:]
		fallthrough
	default:
		if len(args) == 0 {
			printUsage()
			os.Exit(2)
		}
		if len(args) > 1 {
			openBrowser = false
		}
		for _, path := range args {
			Benchmark(ctx, BenchmarkConfigFromPath(path))
		}
	}
}
