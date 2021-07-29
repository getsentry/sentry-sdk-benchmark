package main

import (
	"log"
	"os"
	"time"

	"github.com/sanity-io/litter"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

func x(url string, concurrency int, duration time.Duration) (ok bool) {
	rate := vegeta.Rate{Freq: 500, Per: time.Second}
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    url,
	})
	attacker := vegeta.NewAttacker()
	ch := attacker.Attack(targeter, rate, duration, "")

	var m vegeta.Metrics
	for res := range ch {
		m.Add(res)
	}
	m.Close()

	litter.Dump(m)
	return m.Success == 1.0
}

func main() {
	log.Print("loadgen")
	defer log.Print("Bye!")

	resultPath := os.Getenv("RESULT_PATH")
	if resultPath == "" {
		log.Print("Missing RESULT_PATH env!")
		return
	}

	// wait until ready
	log.Print("Checking web app readiness")
	ready := false
	for i := 0; i < 5; i++ {
		ready = x("http://app:8080/update?query=1", 8, 5*time.Second)
		if ready {
			break
		}
		log.Print("Web app not ready, waiting...")
		time.Sleep(2 * time.Second)
	}
	if !ready {
		panic("Web app not ready")
	}

	// warmup
	log.Print("Warming up web app")
	x("http://app:8080/update?query=100", 256, 15*time.Second)

	// captured test
	log.Print("Testing web app")
	x("http://app:8080/update?query=100", 512, 15*time.Second)

	if err := os.MkdirAll(resultPath, 0777); err != nil {
		panic(err)
	}
}
