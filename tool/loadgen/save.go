package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

type Options struct {
	TargetURL      string        `json:"target_url"`
	CAdvisorURL    string        `json:"cadvisor_url"`
	FakerelayURL   string        `json:"fakerelay_url"`
	Containers     string        `json:"containers"`
	MaxWait        time.Duration `json:"max_wait"`
	WarmupDuration time.Duration `json:"warmup_duration"`
	TestDuration   time.Duration `json:"test_duration"`
	RPS            uint          `json:"rps"`
	Out            string        `json:"out"`
}

// TestResult is the data collected for a test run.
type TestResult struct {
	FirstAppResponse string
	*vegeta.Metrics
	LoadGenResult  []*vegeta.Result       `json:"loadgen_result"`
	Stats          map[string]Stats       `json:"container_stats"`
	RelayMetrics   map[string]interface{} `json:"relay_metrics,omitempty"`
	LoadGenCommand string                 `json:"loadgen_command"`
	Options        Options                `json:"options"`
}

// save writes reports computed from metrics to the output path.
func save(r TestResult, path string) {
	log.Printf("Writing result to %q", path)
	if err := os.MkdirAll(path, 0777); err != nil {
		panic(err)
	}

	// Print text report to stdout, limiting number of bytes to keep program
	// output short. Avoid printing a possibly long list of errors that
	// could hide the path to the complete results written to disk.
	textReporter := vegeta.NewTextReporter(r.Metrics)
	// Note: it doesn't really make sense to write the complete report to a
	// buffer just to possibly truncate it later. And then generate the
	// report again to write it to a file... This is a pragmatic solution,
	// not an efficient one.
	var b bytes.Buffer
	_ = textReporter.Report(&b)
	const max = 1024
	if b.Len() > max {
		b.Truncate(max)
		b.WriteString("... (truncated)\n")
	}
	_, _ = io.Copy(os.Stdout, &b)

	writeReport(textReporter, filepath.Join(path, "report.txt"))
	writeReport(NewJSONReporter(r), filepath.Join(path, "result.json"))
	writeReport(vegeta.NewHDRHistogramPlotReporter(r.Metrics), filepath.Join(path, "histogram.hdr"))
}

// writeReport writes the output of the reporter to the output path.
func writeReport(r vegeta.Reporter, path string) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := r.Report(f); err != nil {
		panic(err)
	}
}

// NewJSONReporter returns a vegeta.Reporter that writes out pretty JSON.
func NewJSONReporter(m interface{}) vegeta.Reporter {
	return func(w io.Writer) error {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	}
}
