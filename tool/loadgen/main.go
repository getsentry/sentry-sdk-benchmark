package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

const target = "http://app:8080"

func main() {
	log.Print("Load Generator")
	defer log.Print("Bye!")
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()

	path := os.Getenv("RESULT_PATH")
	if path == "" {
		panic("Missing RESULT_PATH env var, aborting!")
	}

	waitUntilReady()
	warmUp()
	metrics := test()
	save(metrics, path)
}

// waitUntilReady waits until the target web app is ready to receive traffic.
func waitUntilReady() {
	log.Print("Waiting until web app is ready")
	ready := false
	for i := 0; i < 5; i++ {
		ready = fetch(target+"/update?query=1", 5*time.Second).Success == 1
		if ready {
			break
		}
		log.Print("Web app not ready, waiting...")
		time.Sleep(2 * time.Second)
	}
	if !ready {
		panic("Web app not ready")
	}
}

// warmUp sends some traffic to warm up the target web app, ensuring
// connectivity with the database is established, caches are warm, any JIT has
// taken place, etc.
func warmUp() {
	log.Print("Warming up web app")
	fetch(target+"/update?query=100", 15*time.Second)
}

// test sends the actual test traffic to the target web app and returns the
// collected results.
func test() *vegeta.Metrics {
	log.Print("Testing web app")
	return fetch(target+"/update?query=100", 20*time.Second)
}

// fetch requests the given URL several times for the given duration and with
// the given concurrency level.
func fetch(url string, duration time.Duration) *vegeta.Metrics {
	target := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    url,
	})
	rate := vegeta.Rate{Freq: 100, Per: time.Second}
	attacker := vegeta.NewAttacker()
	ch := attacker.Attack(target, rate, duration, "")

	var m vegeta.Metrics
	for res := range ch {
		m.Add(res)
	}
	m.Close()
	return &m
}

// save writes reports computed from metrics to the output path.
func save(m *vegeta.Metrics, path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		panic(err)
	}
	for _, r := range []struct {
		name string
		fn   func(*vegeta.Metrics) vegeta.Reporter
	}{
		{"txt", vegeta.NewTextReporter},
		{"json", NewJSONReporter},
		// {"hist", vegeta.NewHistogramReporter},
		{"hdr", vegeta.NewHDRHistogramPlotReporter},
	} {
		writeReport(r.fn(m), path+"."+r.name)
	}
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

// NewJSONReporter returns a Reporter that writes out Metrics as pretty JSON.
func NewJSONReporter(m *vegeta.Metrics) vegeta.Reporter {
	return func(w io.Writer) error {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	}
}
