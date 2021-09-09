package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/perf/benchstat"
)

const BenchstatPrefix = "Benchmark"

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

	c := &benchstat.Collection{}

	for key, b := range builders {
		if err := c.AddFile(key, strings.NewReader(b.String())); err != nil {
			panic(err)
		}
	}

	tables := c.Tables()
	var buf bytes.Buffer
	benchstat.FormatText(&buf, tables)
	os.Stdout.Write(buf.Bytes())
}

func addResult(b *strings.Builder, folderPath string) {
	tr := readTestResult(filepath.Join(folderPath, "result.json"))

	// Latencies
	const LatenciesPrefix = BenchstatPrefix + "Latencies"
	writeStat(b, LatenciesPrefix, "Total", "1", floatToString(tr.Latencies.Total.Seconds()), "s")
	writeStat(b, LatenciesPrefix, "Mean", "1", intToString(tr.Latencies.Mean.Milliseconds()), "ms")
	writeStat(b, LatenciesPrefix, "50th", "1", intToString(tr.Latencies.P50.Milliseconds()), "ms")
	writeStat(b, LatenciesPrefix, "90th", "1", intToString(tr.Latencies.P90.Milliseconds()), "ms")
	writeStat(b, LatenciesPrefix, "95th", "1", intToString(tr.Latencies.P95.Milliseconds()), "ms")
	writeStat(b, LatenciesPrefix, "99th", "1", intToString(tr.Latencies.P99.Milliseconds()), "ms")
	writeStat(b, LatenciesPrefix, "Max", "1", intToString(tr.Latencies.Max.Milliseconds()), "ms")
	writeStat(b, LatenciesPrefix, "Min", "1", intToString(tr.Latencies.Min.Milliseconds()), "ms")

	// Other Metrics
	writeStat(b, BenchstatPrefix, "Duration", "1", intToString(tr.Duration.Milliseconds()), "ms")
	writeStat(b, BenchstatPrefix, "Wait", "1", intToString(tr.Wait.Milliseconds()), "ms")
	writeStat(b, BenchstatPrefix, "Requests", "1", uintToString(tr.Requests), "req")
	writeStat(b, BenchstatPrefix, "Rate", "1", floatToString(tr.Rate), "req/s")
	writeStat(b, BenchstatPrefix, "Throughput", "1", floatToString(tr.Throughput), "req/s")
}

func writeStat(b *strings.Builder, prefix, suffix, iterations, value, unit string) {
	b.WriteString(prefix)
	b.WriteString(suffix)
	b.WriteRune(' ')
	b.WriteString(iterations)
	b.WriteRune(' ')
	b.WriteString(value)
	b.WriteRune(' ')
	b.WriteString(unit)
	b.WriteRune('\n')
}

func intToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

func uintToString(i uint64) string {
	return strconv.FormatUint(i, 10)
}

func floatToString(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
