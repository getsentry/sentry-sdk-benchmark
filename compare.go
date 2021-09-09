package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/perf/benchstat"
)

// Compare compares runs from the same platform
func Compare(resultPaths []string) {
	// m maps names like "baseline" and "instrumented" to textual data in
	// the Go Benchmark Data Format. See https://golang.org/issue/14313.
	m := make(map[string][]byte)

	// Walk each directory from resultPaths and collect benchmark data from
	// all result.json files.
	for _, root := range resultPaths {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				panic(err)
			}
			if !d.IsDir() {
				return nil
			}
			defer func() {
				// ignore panics from readTestResult
				_ = recover()
			}()
			tr := readTestResult(filepath.Join(path, "result.json"))
			m[d.Name()] = append(m[d.Name()], toGoBenchFormat(tr)...)
			return nil
		})
	}

	// Sort names to make output deterministic.
	var names []string
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	var c benchstat.Collection
	for _, name := range names {
		c.AddConfig(name, m[name])
	}

	// Print comparison.
	benchstat.FormatText(os.Stdout, c.Tables())
}

func toGoBenchFormat(tr TestResult) []byte {
	var b bytes.Buffer
	writeln := func(name string, value interface{}, unit string) {
		fmt.Fprintf(&b, "Benchmark%s 1 %v %s\n", name, value, unit)
	}
	writeln("LatenciesTotal", tr.Latencies.Total.Seconds(), "s")
	writeln("LatenciesMean", tr.Latencies.Mean.Milliseconds(), "ms")
	writeln("Latencies50th", tr.Latencies.P50.Milliseconds(), "ms")
	writeln("Latencies90th", tr.Latencies.P90.Milliseconds(), "ms")
	writeln("Latencies95th", tr.Latencies.P95.Milliseconds(), "ms")
	writeln("Latencies99th", tr.Latencies.P99.Milliseconds(), "ms")
	writeln("LatenciesMax", tr.Latencies.Max.Milliseconds(), "ms")
	writeln("LatenciesMin", tr.Latencies.Min.Milliseconds(), "ms")
	writeln("Duration", tr.Duration.Milliseconds(), "ms")
	writeln("Wait", tr.Wait.Milliseconds(), "ms")
	writeln("Requests", tr.Requests, "req")
	writeln("Rate", tr.Rate, "req/s")
	writeln("Throughput", tr.Throughput, "req/s")
	return b.Bytes()
}
