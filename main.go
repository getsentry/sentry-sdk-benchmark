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
Usage:
%[1]s [run] PLATFORM [PLATFORM ...]

	Benchmark one or more platforms.

	Examples:
	%[1]s platform/python/django
	%[1]s run platform/javascript/express

%[1]s report RESULT [RESULT ...]

	Print an HTML report summarizing the results of one or more benchmark runs.
	Reports are automatically created after a successful benchmark run.
	This subcommand allows re-generating a report from benchmark result data on demand.

	Examples:
	%[1]s report result/python/django/20210818-082527-tbnfsga
	%[1]s report result/python/django/20210818-*

%[1]s compose APP

	Print a Docker Compose YAML configuration file to run APP and its dependencies.

	Examples:
	%[1]s compose platform/python/django/baseline
	%[1]s compose platform/python/django/instrumented
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
	case "compose":
		args = args[1:]
		if len(args) != 1 {
			printUsage()
			os.Exit(2)
		}
		cfg := BenchmarkConfigFromPlatform(filepath.Dir(args[0]))
		composeFile, _ := Compose(cfg, cfg.Runs[0])
		os.Stdout.Write(composeFile)
		if composeFile[len(composeFile)-1] != '\n' {
			os.Stdout.WriteString("\n")
		}
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
		for _, platform := range args {
			Benchmark(ctx, BenchmarkConfigFromPlatform(platform))
		}
	}
}
