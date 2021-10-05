package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/getsentry/sentry-sdk-benchmark/internal/plot"
	"github.com/getsentry/sentry-sdk-benchmark/internal/std/browser"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

var sdkNameRegex = regexp.MustCompile(`sentry\.([^\s.]+)`)

var reportTemplate = template.Must(template.New("report.html.tmpl").Funcs(reportFuncMap).ParseFiles(filepath.Join("template", "report.html.tmpl")))

var reportCSS []template.CSS
var reportJS []template.HTML

func init() {
	reportCSS = getCSSAssets([]string{"report.css", "dygraph.css"})
	reportJS = getJSAssets([]string{"dygraph.min.js", "script.js"})
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
		ID:        filepath.Base(filepath.Dir(results[0].Path)),
		ReportCSS: reportCSS,
		ReportJS:  reportJS,
		Latency: []Latency{
			{
				Name: "baseline",
				Diff: nil,
			},
		},
	}

	// Extract out baseline as order of run results is unknown
	var baselineResult TestResult
	for _, res := range results {
		if res.Name == "baseline" {
			baselineResult = readTestResult(filepath.Join(res.Path, "result.json"))
		}
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

		if name == "baseline" {
			reportFile.Latency[0].Metrics = tr.Latencies
		} else {
			reportFile.Latency = append(reportFile.Latency, Latency{
				Name:    name,
				Diff:    getLatencyDiff(baselineResult.Latencies, tr.Latencies),
				Metrics: tr.Latencies,
			})
		}

		if math.Round(tr.Throughput) != math.Round(tr.Rate) {
			data.ThroughputDifferent = true
		}

		reportFile.LoadGenOptions = tr.Options

		if len(tr.Errors) > 0 {
			reportFile.HasErrors = true
		}

		for _, r := range tr.LoadGenResult {
			r.Attack = name
			p.Add(r)
		}

		data.TestResult = tr
		data.TestResultJSON = marshalToStr(tr)

		reportFile.Data = append(reportFile.Data, data)
	}

	// FIXME: AppDetails might be different per run. For now, this takes the
	// first non-empty value.
	for _, data := range reportFile.Data {
		sdkInfo := data.TestResult.RelayMetrics.SDKInfo
		var empty SDKInfo
		if sdkInfo != empty {
			reportFile.AppDetails = getAppDetails(results[0].Path, sdkInfo)
			break
		}
	}

	plotData, err := p.GetData()
	if err != nil {
		panic(err)
	}
	// TODO(abhi): have a global list of ids we can refer to.
	// TODO(vladan): make a chart width responsive 100%
	reportFile.LatencyPlot, err = GenerateChart(
		"latencyTimePlot",
		plotData.Data,
		DygraphsOpts{
			Title:       "Latency over Time",
			Labels:      plotData.Labels,
			YLabel:      "Latency (ms)",
			XLabel:      "Seconds elapsed",
			Legend:      "always",
			ShowRoller:  true,
			LogScale:    true,
			StrokeWidth: 1.3,
			Width:       1500,
			RollPeriod:  5,
		},
	)
	if err != nil {
		panic(err)
	}

	reportPath := filepath.Join(filepath.Dir(reportFile.Title), "report.html")

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
	ID        string
	Title     string
	Data      []ResultData
	HasErrors bool

	LatencyPlot template.HTML
	ReportCSS   []template.CSS
	ReportJS    []template.HTML

	AppDetails     AppDetails
	LoadGenOptions Options
	Latency        []Latency
}

type AppDetails struct {
	Language   string
	Framework  string
	SdkName    string
	SdkVersion string
}

type ResultData struct {
	Name                string
	HDR                 string
	TestResult          TestResult
	TestResultJSON      string
	ThroughputDifferent bool
}

type RelayMetrics struct {
	Requests      int     `json:"requests"`
	FirstRequest  string  `json:"first_request"`
	SDKInfo       SDKInfo `json:"sdk"`
	BytesReceived int     `json:"bytes_received"`
}

type SDKInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Latency struct {
	Name    string                `json:"name"`
	Diff    *LatencyDiff          `json:"diff,omitempty"`
	Metrics vegeta.LatencyMetrics `json:"metrics"`
}

// LatencyDiff stores the percentage difference between
// two vegeta.LatencyMetrics structs
type LatencyDiff struct {
	// Total is the total latency sum of all requests in an attack.
	Total float64 `json:"total"`
	// Mean is the mean request latency.
	Mean float64 `json:"mean"`
	// P50 is the 50th percentile request latency.
	P50 float64 `json:"50th"`
	// P90 is the 90th percentile request latency.
	P90 float64 `json:"90th"`
	// P95 is the 95th percentile request latency.
	P95 float64 `json:"95th"`
	// P99 is the 99th percentile request latency.
	P99 float64 `json:"99th"`
	// Max is the maximum observed request latency.
	Max float64 `json:"max"`
	// Min is the minimum observed request latency.
	Min float64 `json:"min"`
}

// START copied from ./tool/loadgen

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
	LoadGenResult  []*vegeta.Result `json:"loadgen_result"`
	Stats          map[string]Stats `json:"container_stats"`
	RelayMetrics   RelayMetrics     `json:"relay_metrics,omitempty"`
	LoadGenCommand string           `json:"loadgen_command"`
	Options        Options          `json:"options"`
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

func getCSSAssets(paths []string) []template.CSS {
	t := make([]template.CSS, len(paths))
	for i, p := range paths {
		t[i] = template.CSS(readBytes(filepath.Join("template", "css", p)))
	}
	return t
}

func getJSAssets(paths []string) []template.HTML {
	t := make([]template.HTML, len(paths))
	for i, p := range paths {
		js := template.JS(readBytes(filepath.Join("template", "js", p)))
		t[i] = template.HTML("<script>" + js + "</script>")
	}
	return t
}

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
	tr.FirstAppResponse = formatHTTP(tr.FirstAppResponse)
	tr.RelayMetrics.FirstRequest = formatHTTP(tr.RelayMetrics.FirstRequest)
	return tr
}

func marshalToStr(t interface{}) string {
	j, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		panic(err)
	}

	return string(j)
}

// From https://yourbasic.org/golang/formatting-byte-size-to-human-readable-format/
func byteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

var reportFuncMap = template.FuncMap{
	"round": func(t time.Duration) time.Duration {
		if t.Round(time.Second) > 0 {
			return t.Truncate(10 * time.Millisecond)
		}
		return t.Truncate(10 * time.Microsecond)
	},
	"byteFormat": func(b int64) string {
		return byteCountSI(b)
	},
	"byteFormatUnsigned": func(b uint64) string {
		return byteCountSI(int64(b))
	},
	"numRequests": func(rps uint, d time.Duration) uint {
		return uint(d.Seconds()) * rps
	},
	"percentDiffUInt": func(before, after uint64) float64 {
		b := float64(before)
		a := float64(after)

		p := ((a - b) / b) * 100
		return math.Round(p*100) / 100
	},
}

func formatSDKName(n string) string {
	match := sdkNameRegex.FindString(n)
	if match == "" {
		return n
	}
	return strings.ReplaceAll(match, ".", "-")
}

func getAppDetails(path string, sdkInfo SDKInfo) AppDetails {
	pathList := strings.Split(path, string(filepath.Separator))
	return AppDetails{
		Language:   pathList[1],
		Framework:  pathList[2],
		SdkName:    formatSDKName(sdkInfo.Name),
		SdkVersion: sdkInfo.Version,
	}
}

func getLatencyDiff(baseline, final vegeta.LatencyMetrics) *LatencyDiff {
	return &LatencyDiff{
		Total: percentDiff(baseline.Total, final.Total),
		Mean:  percentDiff(baseline.Mean, final.Mean),
		P50:   percentDiff(baseline.P50, final.P50),
		P90:   percentDiff(baseline.P90, final.P90),
		P95:   percentDiff(baseline.P95, final.P95),
		P99:   percentDiff(baseline.P99, final.P99),
		Max:   percentDiff(baseline.Max, final.Max),
		Min:   percentDiff(baseline.Min, final.Min),
	}
}

func percentDiff(start, final time.Duration) float64 {
	s := start.Seconds()
	f := final.Seconds()

	p := ((f - s) / s) * 100
	return math.Round(p*100) / 100
}

// formatHTTP takes a raw HTTP 1.x request or response and pretty-prints JSON
// bodies.
func formatHTTP(b string) string {
	var s strings.Builder
	bodyStart := strings.Index(b, "\r\n\r\n") + 4
	if bodyStart < 4 {
		return b
	}
	s.WriteString(b[:bodyStart])
	body, err := jsonIndent([]byte(b[bodyStart:]))
	if err != nil {
		return b
	}
	s.Write(body)
	return s.String()
}

// jsonIndent is similar to json.Indent but can deal with a stream of JSON
// values.
func jsonIndent(src []byte) ([]byte, error) {
	var w bytes.Buffer
	dec := json.NewDecoder(bytes.NewReader(src))
	enc := json.NewEncoder(&w)
	enc.SetIndent("", "  ")
	for dec.More() {
		var m json.RawMessage
		err := dec.Decode(&m)
		if err != nil {
			return nil, err
		}
		err = enc.Encode(m)
		if err != nil {
			return nil, err
		}
	}
	return w.Bytes(), nil
}
