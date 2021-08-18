package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

var summaryTemplate = template.Must(template.ParseFiles(filepath.Join("template", "summary.html.tmpl")))

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

	var b bytes.Buffer
	err := summaryTemplate.Execute(&b, summaryFile)
	if err != nil {
		panic(err)
	}

	summaryPath := filepath.Join(summaryFile.Title, "summary.html")
	fmt.Printf("Generating benchmark summary at %s", summaryPath)
	if err := os.WriteFile(summaryPath, b.Bytes(), 0666); err != nil {
		panic(err)
	}

	cmd := exec.Command(
		"open",
		summaryPath,
	)
	if err := cmd.Run(); err != nil {
		panic(err)
	}
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
	Stats map[string]Stats `json:"container_stats"`
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
