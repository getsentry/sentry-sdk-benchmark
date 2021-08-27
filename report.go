package main

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"

	"github.com/getsentry/sentry-sdk-benchmark/internal/std/browser"
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

func Report(s []string) {
	if len(s) != 1 {
		panic("Reporting on multiple results is not supported yet")
	}
	report([]*RunResult{
		{
			Name: "baseline",
			Path: filepath.Join(s[0], "baseline"),
		},
		{
			Name: "instrumented",
			Path: filepath.Join(s[0], "instrumented"),
		},
	})
}

func report(results []*RunResult) {
	reportFile := ReportFile{
		Data: make([]ReportFileData, 2),
	}

	for i, res := range results {
		folderPath := filepath.Dir(res.Path)
		name := res.Name

		if i == 0 {
			reportFile.Title = folderPath
		}

		reportFile.Data[i].Name = name
		reportFile.Data[i].HDR = string(readBytes(filepath.Join(folderPath, name+".hdr")))
		tr := readTestResult(filepath.Join(folderPath, name+".json"))

		if tr.RelayMetrics != nil {
			reportFile.RelayMetrics = tr.RelayMetrics
		}

		reportFile.Data[i].TestResult = tr
	}

	reportPath := filepath.Join(reportFile.Title, "report.html")
	f, err := os.Create(reportPath)
	if err != nil {
		panic(err)
	}

	log.Printf("Generating benchmark report at %s", reportPath)
	if err := reportTemplate.Execute(f, reportFile); err != nil {
		panic(err)
	}

	browser.Open(reportPath)
}

type ReportFile struct {
	Title        string
	Data         []ReportFileData
	RelayMetrics map[string]interface{}
}

type ReportFileData struct {
	Name       string
	HDR        string
	TestResult TestResult
}

// START copied from ./tool/loadgen

type TestResult struct {
	*vegeta.Metrics
	Stats        map[string]Stats       `json:"container_stats"`
	RelayMetrics map[string]interface{} `json:"relay_metrics,omitempty"`
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
