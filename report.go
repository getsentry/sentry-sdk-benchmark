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

var summaryTemplate = template.Must(template.ParseFiles(filepath.Join("template", "summary.html.tmpl")))

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
	summaryFile := SummaryFile{
		Data: make([]SummaryFileData, 2),
	}

	for i, res := range results {
		folderPath := filepath.Dir(res.Path)
		name := res.Name

		if i == 0 {
			summaryFile.Title = folderPath
		}
		summaryFile.Data[i].Name = name
		summaryFile.Data[i].HDR = read(filepath.Join(folderPath, name+".hdr"))
		summaryFile.Data[i].JSON = mustJSONUnmarshal(filepath.Join(folderPath, name+".json"))
	}

	summaryPath := filepath.Join(summaryFile.Title, "summary.html")
	f, err := os.Create(summaryPath)
	if err != nil {
		panic(err)
	}

	log.Printf("Generating benchmark summary at %s", summaryPath)
	if err := summaryTemplate.Execute(f, summaryFile); err != nil {
		panic(err)
	}

	browser.Open(summaryPath)
}

type SummaryFile struct {
	Title string
	Data  []SummaryFileData
}

type SummaryFileData struct {
	Name string
	HDR  string
	JSON TestResult
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

func read(path string) string {
	file, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	return string(file)
}

func mustJSONUnmarshal(path string) (out TestResult) {
	if err := json.Unmarshal([]byte(read(path)), &out); err != nil {
		panic(err)
	}
	return out
}
