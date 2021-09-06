package main

import (
	"fmt"
	"log"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

// fetch makes rps requests per second to fetch the given URL for the given
// duration and returns metrics.
func fetch(url string, rps uint, duration time.Duration, opts ...func(*vegeta.Attacker)) *vegeta.Metrics {
	target := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    url,
	})
	rate := vegeta.Rate{Freq: int(rps), Per: time.Second}
	attacker := vegeta.NewAttacker(opts...)
	ch := attacker.Attack(target, rate, duration, "")

	var m vegeta.Metrics
	for res := range ch {
		m.Add(res)
	}
	m.Close()
	return &m
}

// waitUntilReady waits until the target web app is ready to receive traffic.
func waitUntilReady(url string, maxWait time.Duration) {
	if maxWait == 0 {
		log.Print("Assuming target is ready")
		return
	}

	start := time.Now()
	deadline := start.Add(maxWait)
	const maxSleep = 10 * time.Second

	for i := 0; ; i++ {
		log.Print("Waiting until target is ready")

		metrics := fetch(url, 1, time.Second, vegeta.Timeout(10*time.Second))

		if n := metrics.StatusCodes["404"]; n > 0 {
			panic(fmt.Errorf("bad target %q: got %d responses with status code 404", url, n))
		}
		if ready := metrics.Success == 1.0; ready {
			return
		}
		// exponential back off starting at 500ms, capped at maxSleep
		sleep := (1 << i) * 500 * time.Millisecond
		if sleep > maxSleep {
			sleep = maxSleep
		}
		if time.Now().Add(sleep).After(deadline) {
			_ = vegeta.NewTextReporter(metrics).Report(log.Writer())
			panic(fmt.Errorf("target not ready after %v", time.Since(start)))
		}
		log.Printf("Backing off for %v", sleep)
		time.Sleep(sleep)
	}
}

// warmUp sends traffic to warm up the target web app, ensuring connectivity
// with the database is established, caches are warm, any JIT has taken place,
// etc.
func warmUp(url string, rps uint, d time.Duration) {
	log.Printf("Warming up target for %v", d)
	fetch(url, rps, d)
}

// test sends test traffic to the target web app and returns metrics.
func test(url string, rps uint, d time.Duration) *vegeta.Metrics {
	log.Printf("Testing target for %v", d)
	return fetch(url, rps, d)
}
