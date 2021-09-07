package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lmsgprefix)
	log.SetPrefix("[loadgen] ")

	defer func() {
		if err := recover(); err != nil {
			_, file, line, ok := runtime.Caller(2)
			if !ok {
				panic(err)
			}
			log.Fatalf("Failure: %s:%d: %s", file, line, err)
		}
	}()

	var (
		targetURL, cAdvisorURL, fakerelayURL string
		containers                           string
		maxWait                              time.Duration
		warmupDuration, testDuration         time.Duration
		rps                                  uint
		out                                  string
	)
	flag.StringVar(&targetURL, "target", "", "target `URL` (example \"http://app:8080/update?queries=10\") (required)")
	flag.StringVar(&cAdvisorURL, "cadvisor", "", "cAdvisor root `URL` (example \"http://cadvisor:8080\")")
	flag.StringVar(&fakerelayURL, "fakerelay", "", "fakerelay root `URL` (example \"http://relay:5000\")")
	flag.StringVar(&containers, "containers", "", "comma-separated list of container `names` to monitor with cAdvisor")
	flag.DurationVar(&maxWait, "maxwait", 30*time.Second, "max wait until target is ready")
	flag.DurationVar(&warmupDuration, "warmup", 15*time.Second, "warmup duration")
	flag.DurationVar(&testDuration, "test", 30*time.Second, "test duration")
	flag.UintVar(&rps, "rps", 10, "requests per second")
	flag.StringVar(&out, "out", filepath.Join(os.TempDir(), "loadgen", "result", time.Now().Format("20060102-150405")), "output path")
	flag.Parse()

	if targetURL == "" {
		fmt.Fprintln(flag.CommandLine.Output(), "flag -target is required")
		flag.Usage()
		os.Exit(1)
	}

	if cAdvisorURL != "" && containers == "" {
		panic("flag -containers is required when -cadvisor is provided")
	}
	if strings.Contains(containers, ",") {
		panic("not implemented: only one container name supported")
	}
	containerName := containers

	log.Printf("Target is %q", targetURL)

	waitUntilReady(targetURL, maxWait)
	warmUp(targetURL, rps, warmupDuration)

	stats := Stats{}
	if cAdvisorURL != "" {
		stats.Before = containerStats(cAdvisorURL, containerName)
	}

	r := test(targetURL, rps, testDuration)
	metrics := r.Metrics

	if cAdvisorURL != "" {
		stats.After = containerStats(cAdvisorURL, containerName)
		// Note: potential overflow ignored for simplicity
		stats.Difference.Duration = stats.After.Timestamp.Sub(stats.Before.Timestamp)
		stats.Difference.MemoryMaxUsageBytes = int64(stats.After.MemoryMaxUsageBytes - stats.Before.MemoryMaxUsageBytes)
		stats.Difference.CPUUsageUser = int64(stats.After.CPUUsageUser - stats.Before.CPUUsageUser)
		stats.Difference.CPUUsageSystem = int64(stats.After.CPUUsageSystem - stats.Before.CPUUsageSystem)
		stats.Difference.CPUUsageTotal = int64(stats.After.CPUUsageTotal - stats.Before.CPUUsageTotal)
	}

	result := TestResult{
		FirstAppResponse: r.FirstResponse,
		Metrics:          metrics,
		LoadGenOptions: struct{ Command []string }{
			Command: os.Args,
		},
		Stats: map[string]Stats{"app": stats},
	}
	if fakerelayURL != "" {
		result.RelayMetrics = relayMetrics(fakerelayURL)
	}

	save(result, out)

	log.Print("Success")
}
