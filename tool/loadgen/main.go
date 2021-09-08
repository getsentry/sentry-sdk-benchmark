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

	log.Printf("Target is %q", targetURL)

	waitUntilReady(targetURL, maxWait)
	warmUp(targetURL, rps, warmupDuration)

	stats := make(map[string]Stats)
	if cAdvisorURL != "" {
		for _, containerName := range strings.Split(containers, ",") {
			imageName := strings.Split(containerName, "-")[0]

			stats[imageName] = Stats{
				Before: containerStats(cAdvisorURL, containerName),
			}
		}
	}

	r := test(targetURL, rps, testDuration)
	metrics := r.Metrics

	if cAdvisorURL != "" {
		for _, containerName := range strings.Split(containers, ",") {
			imageName := strings.Split(containerName, "-")[0]

			after := containerStats(cAdvisorURL, containerName)
			before := stats[imageName].Before

			stats[imageName] = Stats{
				Before: before,
				After:  after,
				Difference: ContainerStatsDifference{
					Duration:            after.Timestamp.Sub(before.Timestamp),
					MemoryMaxUsageBytes: int64(after.MemoryMaxUsageBytes - before.MemoryMaxUsageBytes),
					CPUUsageUser:        int64(after.CPUUsageUser - before.CPUUsageUser),
					CPUUsageSystem:      int64(after.CPUUsageSystem - before.CPUUsageSystem),
					CPUUsageTotal:       int64(after.CPUUsageTotal - before.CPUUsageTotal),
				},
			}
		}
	}

	result := TestResult{
		FirstAppResponse: r.FirstResponse,
		Metrics:          metrics,
		LoadGenCommand:   strings.Join(os.Args, " "),
		Stats:            stats,
	}
	if fakerelayURL != "" {
		result.RelayMetrics = relayMetrics(fakerelayURL)
	}

	save(result, out)

	log.Print("Success")
}
