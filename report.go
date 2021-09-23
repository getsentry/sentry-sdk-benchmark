package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
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

var funcMap = template.FuncMap{
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
	"formatSDKName": formatSDKName,
}

var reportTemplate = template.Must(template.New("report.html.tmpl").Funcs(funcMap).ParseFiles(filepath.Join("template", "report.html.tmpl")))

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

		if len(tr.Errors) > 0 {
			reportFile.HasErrors = true
		}

		for _, r := range tr.LoadGenResult {
			r.Attack = name
			p.Add(r)
		}

		data.TestResult = tr
		data.TestResultJSON = marshalToStr(tr)

		if tr.RelayMetrics != nil {
			setRelayData(&reportFile, tr.RelayMetrics)
		}

		reportFile.Data = append(reportFile.Data, data)
	}

	reportFile.Configuration = getConfiguration(reportFile.Title, reportFile.RelayMetrics)

	plotData, err := p.GetData()
	if err != nil {
		panic(err)
	}

	// TODO(abhi): have a global list of ids we can refer to.
	latencyPlot, err := GenerateChart("latencyTimePlot", plotData.Data, DygraphsOpts{
		Title:       plotData.Title,
		Labels:      plotData.Labels,
		YLabel:      "Latency (ms)",
		XLabel:      "Seconds elapsed",
		Legend:      "always",
		ShowRoller:  true,
		LogScale:    true,
		StrokeWidth: 1.3,
	})
	if err != nil {
		panic(err)
	}

	reportFile.LatencyPlot = latencyPlot

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
	HasErrors           bool
	LatencyPlot         template.HTML
	ReportCSS           []template.CSS
	ReportJS            []template.HTML
	Configuration       Configuration
}

type Configuration struct {
	Language   string
	Framework  string
	SdkName    string
	SdkVersion string
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

func setRelayData(f *ReportFile, relayMetrics map[string]interface{}) {
	f.RelayMetrics = relayMetrics

	firstReq, ok := relayMetrics["first_request"]
	if !ok {
		return
	}

	var h strings.Builder
	var e strings.Builder

	reqArr := strings.Split(firstReq.(string), "\n")
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

	f.FirstRequestHeaders = h.String()
	f.FirstRequestEnv = e.String()
}

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
	return tr
}

func marshalToStr(t interface{}) string {
	j, err := json.Marshal(t)
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

func formatSDKName(n string) string {
	match := sdkNameRegex.FindString(n)
	if match == "" {
		return n
	}
	return strings.ReplaceAll(match, ".", "-")
}

func getConfiguration(path string, relayMetrics map[string]interface{}) Configuration {
	pathList := strings.Split(path, string(filepath.Separator))
	sdk := make(map[string]string)

	for k, v := range relayMetrics {
		switch v := v.(type) {
		case map[string]interface{}:
			if k == "sdk" {
				for n, i := range v {
					switch i := i.(type) {
					case string:
						sdk[n] = i
					}
				}
			}
		}
	}

	return Configuration{
		Language:   pathList[1],
		Framework:  pathList[2],
		SdkName:    formatSDKName(sdk["name"]),
		SdkVersion: sdk["version"],
	}
}
