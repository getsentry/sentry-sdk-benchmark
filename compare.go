package main

import (
	"bytes"
	"fmt"
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
			panic(fmt.Errorf("[compare] %s is not a valid path", path))
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

	c := &benchstat.Collection{
		Alpha:      0.05,
		AddGeoMean: false,
		DeltaTest:  benchstat.UTest,
	}

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
	writeStat(b, LatenciesPrefix, "Total", "1", tr.Latencies.Total.Milliseconds())
	writeStat(b, LatenciesPrefix, "Mean", "1", tr.Latencies.Mean.Milliseconds())
	writeStat(b, LatenciesPrefix, "50th", "1", tr.Latencies.P50.Milliseconds())
	writeStat(b, LatenciesPrefix, "90th", "1", tr.Latencies.P90.Milliseconds())
	writeStat(b, LatenciesPrefix, "95th", "1", tr.Latencies.P95.Milliseconds())
	writeStat(b, LatenciesPrefix, "95th", "1", tr.Latencies.P95.Milliseconds())
	writeStat(b, LatenciesPrefix, "99th", "1", tr.Latencies.P99.Milliseconds())
	writeStat(b, LatenciesPrefix, "Max", "1", tr.Latencies.Max.Milliseconds())
	writeStat(b, LatenciesPrefix, "Min", "1", tr.Latencies.Min.Milliseconds())

	// Other Metrics
	writeStat(b, BenchstatPrefix, "Duration", "1", tr.Duration.Milliseconds())
	writeStat(b, BenchstatPrefix, "Wait", "1", tr.Wait.Milliseconds())
	writeStat(b, BenchstatPrefix, "Requests", "1", int64(tr.Requests))
	writeStat(b, BenchstatPrefix, "Rate", "1", int64(tr.Rate))
	writeStat(b, BenchstatPrefix, "Throughput", "1", int64(tr.Throughput))
}

func writeStat(b *strings.Builder, prefix, suffix, iterations string, value int64) {
	b.WriteString(prefix)
	b.WriteString(suffix)
	b.WriteRune(' ')
	b.WriteString(iterations)
	b.WriteRune(' ')
	b.WriteString(strconv.FormatInt(value, 10))
	b.WriteRune(' ')
	b.WriteString("ms")
	b.WriteRune('\n')
}
