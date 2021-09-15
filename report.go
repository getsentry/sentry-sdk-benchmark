package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/getsentry/sentry-sdk-benchmark/internal/std/browser"
	vegeta "github.com/tsenart/vegeta/v12/lib"
	"github.com/tsenart/vegeta/v12/lib/plot"
)

var funcMap = template.FuncMap{
	"round": func(t time.Duration) time.Duration {
		if t.Round(time.Second) > 0 {
			return t.Truncate(10 * time.Millisecond)
		}
		return t.Truncate(10 * time.Microsecond)
	},
}
var reportTemplate = template.Must(template.New("report.html.tmpl").Funcs(funcMap).ParseFiles(filepath.Join("template", "report.html.tmpl")))

var reportCSS template.CSS
var reportJS template.JS

func init() {
	reportCSS = template.CSS(readBytes(filepath.Join("template", "report.css")))
	reportJS = template.JS(readBytes(filepath.Join("template", "script.js")))
}

// Report generates an HTML report summarizing the results of one or more benchmark runs.
//
// Must be called with 1 or more valid result paths.
func Report(s []string) {
	entries, err := os.ReadDir(s[0])
	if err != nil {
		panic(err)
	}

	// Generate run results from folder names
	var runResults []*RunResult
	hasMultipleResults := len(s) > 1

	for _, resultPath := range s {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			path := filepath.Join(resultPath, name)

			// If there is multiple results, we need to uniquely identify them by more than just
			// their name (baseline, instrumented), so we rely on the entire folder path.
			if hasMultipleResults {
				name = path
			}

			runResults = append(runResults, &RunResult{
				Name: name,
				Path: path,
			})
		}
	}

	if len(runResults) == 0 {
		panic(fmt.Errorf("no valid results in: %s", s))
	}

	report(runResults)
}

func report(results []*RunResult) {
	reportFile := ReportFile{
		ReportCSS: reportCSS,
		ReportJS:  reportJS,
	}

	p := plot.New()

	for i, res := range results {
		folderPath := res.Path
		name := res.Name

		if i == 0 {
			reportFile.Title = folderPath
		}

		var data ResultData

		data.Name = name
		data.HDR = string(readBytes(filepath.Join(folderPath, "histogram.hdr")))
		tr := readTestResult(filepath.Join(folderPath, "result.json"))

		for _, r := range tr.LoadGenResult {
			r.Attack = name
			p.Add(r)
		}

		if tr.RelayMetrics != nil {
			reportFile.RelayMetrics = tr.RelayMetrics

			reqArr := strings.Split(reportFile.RelayMetrics["first_request"].(string), "\n")

			var h strings.Builder
			var e strings.Builder

			for _, r := range reqArr {
				if r == "" || r == "\r" {
					continue
				}

				// hack to check if it's an envelope item
				if r[0] == '{' {
					e.WriteString(r)
					e.WriteRune('\n')
				} else {
					h.WriteString(r)
				}
			}

			reportFile.FirstRequestHeaders = h.String()
			reportFile.FirstRequestEnv = e.String()
		}

		data.TestResult = tr

		resJSON, err := json.Marshal(tr)
		if err != nil {
			panic(err)
		}

		data.TestResultJSON = string(resJSON)

		reportFile.Data = append(reportFile.Data, data)
	}

	var b bytes.Buffer
	p.WriteTo(&b)
	reportFile.LatencyPlot = template.HTML(b.String())

	var reportPath string
	if len(results) > 2 {
		reportPath = filepath.Join(filepath.Dir(filepath.Dir(reportFile.Title)), "report.html")
	} else {
		reportPath = filepath.Join(filepath.Dir(reportFile.Title), "report.html")
	}

	f, err := os.Create(reportPath)
	if err != nil {
		panic(err)
	}

	log.Printf("Writing benchmark report to %q", reportPath)
	if err := reportTemplate.Execute(f, reportFile); err != nil {
		panic(err)
	}

	if openBrowser {
		browser.Open(reportPath)
	}
}

type ReportFile struct {
	Title               string
	Data                []ResultData
	RelayMetrics        map[string]interface{}
	FirstRequestHeaders string
	FirstRequestEnv     string
	LatencyPlot         template.HTML
	ReportCSS           template.CSS
	ReportJS            template.JS
}

type ResultData struct {
	Name           string
	HDR            string
	TestResult     TestResult
	TestResultJSON string
}

// START copied from ./tool/loadgen

type TestResult struct {
	FirstAppResponse string
	*vegeta.Metrics
	LoadGenResult  []*vegeta.Result       `json:"loadgen_result"`
	Stats          map[string]Stats       `json:"container_stats"`
	RelayMetrics   map[string]interface{} `json:"relay_metrics,omitempty"`
	LoadGenCommand string                 `json:"loadgen_command"`
}

type Stats struct {
	Before     ContainerStats           `json:"before"`
	After      ContainerStats           `json:"after"`
	Difference ContainerStatsDifference `json:"difference"`
}

type ContainerStats struct {
	Timestamp           time.Time `json:"timestamp"`
	MemoryMaxUsageBytes uint64    `json:"memory_max_usage_bytes"`
	CPUUsageUser        uint64    `json:"cpu_usage_user"`
	CPUUsageSystem      uint64    `json:"cpu_usage_system"`
	CPUUsageTotal       uint64    `json:"cpu_usage_total"`
}

type ContainerStatsDifference struct {
	Duration            time.Duration `json:"duration"`
	MemoryMaxUsageBytes int64         `json:"memory_max_usage_bytes"`
	CPUUsageUser        int64         `json:"cpu_usage_user"`
	CPUUsageSystem      int64         `json:"cpu_usage_system"`
	CPUUsageTotal       int64         `json:"cpu_usage_total"`
}

// END copied from ./tool/loadgen

func readBytes(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}

func readTestResult(path string) (tr TestResult) {
	if err := json.Unmarshal(readBytes(path), &tr); err != nil {
		panic(err)
	}
	return tr
}
