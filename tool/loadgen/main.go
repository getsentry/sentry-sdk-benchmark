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

	var options Options
	flag.StringVar(&options.TargetURL, "target", "", "target `URL` (example \"http://app:8080/update?queries=10\") (required)")
	flag.StringVar(&options.CAdvisorURL, "cadvisor", "", "cAdvisor root `URL` (example \"http://cadvisor:8080\")")
	flag.StringVar(&options.FakerelayURL, "fakerelay", "", "fakerelay root `URL` (example \"http://relay:5000\")")
	flag.StringVar(&options.Containers, "containers", "", "comma-separated list of container `names` to monitor with cAdvisor")
	flag.DurationVar(&options.MaxWait, "maxwait", 30*time.Second, "max wait until target is ready")
	flag.DurationVar(&options.WarmupDuration, "warmup", 15*time.Second, "warmup duration")
	flag.DurationVar(&options.TestDuration, "test", 30*time.Second, "test duration")
	flag.UintVar(&options.RPS, "rps", 10, "requests per second")
	flag.StringVar(&options.Out, "out", filepath.Join(os.TempDir(), "loadgen", "result", time.Now().Format("20060102-150405")), "output path")
	flag.Parse()

	if options.TargetURL == "" {
		fmt.Fprintln(flag.CommandLine.Output(), "flag -target is required")
		flag.Usage()
		os.Exit(1)
	}

	if options.CAdvisorURL != "" && options.Containers == "" {
		panic("flag -containers is required when -cadvisor is provided")
	}

	log.Printf("Target is %q", options.TargetURL)

	waitUntilReady(options.TargetURL, options.MaxWait)
	warmUp(options.TargetURL, options.RPS, options.WarmupDuration)

	stats := make(map[string]Stats)
	if options.CAdvisorURL != "" {
		for _, containerName := range strings.Split(options.Containers, ",") {
			imageName := strings.Split(containerName, "-")[0]

			stats[imageName] = Stats{
				Before: containerStats(options.CAdvisorURL, containerName),
			}
		}
	}

	r := test(options.TargetURL, options.RPS, options.TestDuration)
	metrics := r.Metrics

	if options.CAdvisorURL != "" {
		for _, containerName := range strings.Split(options.Containers, ",") {
			imageName := strings.Split(containerName, "-")[0]

			after := containerStats(options.CAdvisorURL, containerName)
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
		LoadGenResult:    r.Res,
		Metrics:          metrics,
		LoadGenCommand:   strings.Join(os.Args, " "),
		Stats:            stats,
		Options:          options,
	}
	if options.FakerelayURL != "" {
		result.RelayMetrics = relayMetrics(options.FakerelayURL)
	}

	save(result, options.Out)

	log.Print("Success")
}
