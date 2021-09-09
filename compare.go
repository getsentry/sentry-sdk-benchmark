package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/perf/benchstat"
)

// Compare compares runs from the same platform
func Compare(s []string) {
	builders := make(map[string]*strings.Builder)
	for _, path := range s {
		info, err := os.Stat(path)
		if err != nil {
			panic(err)
		}

		if !info.Mode().IsDir() {
			continue
		}

		files, err := ioutil.ReadDir(path)
		if err != nil {
			panic(err)
		}

		for _, f := range files {
			if f.IsDir() {
				name := f.Name()
				p := filepath.Join(path, name)
				if b, ok := builders[name]; ok {
					addResult(b, p)
				} else {
					var sb strings.Builder
					builders[name] = &sb
					addResult(&sb, p)
				}
			}
		}
	}

	// Sort names to make output deterministic.
	var names []string
	for name := range builders {
		names = append(names, name)
	}
	sort.Strings(names)
	var c benchstat.Collection
	for _, name := range names {
		file := strings.NewReader(builders[name].String())
		if err := c.AddFile(name, file); err != nil {
			panic(err)
		}
	}

	// Print comparison.
	benchstat.FormatText(os.Stdout, c.Tables())
}

func addResult(w io.Writer, folderPath string) {
	tr := readTestResult(filepath.Join(folderPath, "result.json"))

	// Latencies
	writeStat(w, "LatenciesTotal", tr.Latencies.Total.Seconds(), "s")
	writeStat(w, "LatenciesMean", tr.Latencies.Mean.Milliseconds(), "ms")
	writeStat(w, "Latencies50th", tr.Latencies.P50.Milliseconds(), "ms")
	writeStat(w, "Latencies90th", tr.Latencies.P90.Milliseconds(), "ms")
	writeStat(w, "Latencies95th", tr.Latencies.P95.Milliseconds(), "ms")
	writeStat(w, "Latencies99th", tr.Latencies.P99.Milliseconds(), "ms")
	writeStat(w, "LatenciesMax", tr.Latencies.Max.Milliseconds(), "ms")
	writeStat(w, "LatenciesMin", tr.Latencies.Min.Milliseconds(), "ms")

	// Other Metrics
	writeStat(w, "Duration", tr.Duration.Milliseconds(), "ms")
	writeStat(w, "Wait", tr.Wait.Milliseconds(), "ms")
	writeStat(w, "Requests", tr.Requests, "req")
	writeStat(w, "Rate", tr.Rate, "req/s")
	writeStat(w, "Throughput", tr.Throughput, "req/s")
}

func writeStat(w io.Writer, name string, value interface{}, unit string) {
	fmt.Fprintf(w, "Benchmark%s 1 %v %s\n", name, value, unit)
}
