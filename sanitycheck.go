package main

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
)

// sanityCheck panics if r fails to match expectations.
func sanityCheck(r []ResultData) {
	if len(r) == 0 {
		panic("no results")
	}
	for _, rr := range r {
		sanityCheckOne(rr)
	}
}

func sanityCheckOne(r ResultData) {
	log.Print("Sanity check: ", r.Name)

	sanityCheckTargetApp(r)

	sanityCheckLoadGenerator(r)

	switch r.Name {
	case "baseline":
		sanityCheckFakeRelayBaseline(r)
	case "instrumented":
		sanityCheckFakeRelayInstrumented(r)
	case "opentelemetry":
		sanityCheckFakeRelayOpenTelemetry(r)
	default:
		panic(fmt.Errorf("unexpected name: %q", r.Name))
	}

	log.Print("Sanity check: OK")
}

func sanityCheckTargetApp(r ResultData) {
	i := strings.Index(r.TestResult.FirstAppResponse, "\r\n\r\n")
	if i < 0 {
		panic(fmt.Errorf("invalid HTTP response: %q", r.TestResult.FirstAppResponse))
	}
	body := r.TestResult.FirstAppResponse[i:]
	dec := json.NewDecoder(strings.NewReader(body))
	var payload []struct {
		ID           int
		RandomNumber int
	}
	if err := dec.Decode(&payload); err != nil {
		panic(err)
	}
	if len(payload) != 10 {
		panic(fmt.Errorf("first app response returned %d items, want 10", len(payload)))
	}
	last := payload[0]
	for i, item := range payload[1:] {
		if item == last {
			panic(fmt.Errorf("repeated payload item %d: %+v", i, item))
		}
	}
}

func sanityCheckLoadGenerator(r ResultData) {
	if r.ThroughputDifferent {
		panic("unexpected throughput")
	}
	if n := len(r.TestResult.Metrics.Errors); n > 0 {
		panic(fmt.Errorf("load generator observed %d errors", n))
	}
}

func sanityCheckFakeRelayBaseline(r ResultData) {
	var empty RelayMetrics
	if m := r.TestResult.RelayMetrics; m != empty {
		panic(fmt.Errorf("unexpected relay metrics: %q", m))
	}
}

func sanityCheckFakeRelayInstrumented(r ResultData) {
	m := r.TestResult.RelayMetrics
	if got := m.BytesReceived; got <= 0 {
		panic(fmt.Errorf("fakerelay got %d bytes, want >0", got))
	}
	// TODO: we cannot assert on the exact number of requests
	// because r.TestResult.Requests is the number of requests sent
	// during the test, while m.Requests is the number of requests
	// received by fakerelay including the initial readiness check
	// and warmup phases.
	if got, want := m.Requests, int(r.TestResult.Requests); got < want {
		panic(fmt.Errorf("fakerelay got %d requests, want >=%d", got, want))
	}
	fr := m.FirstRequest
	if !strings.HasPrefix(fr, "POST /api/1/envelope") {
		panic(fmt.Errorf("fakerelay bad first request"))
	}
	re := regexp.MustCompile(`  "transaction": "([^"]+)",`)
	log.Printf("transaction: %s", re.FindStringSubmatch(fr)[1])
	re = regexp.MustCompile(`(?m)"spans": \[[^]]+\]`)
	spansField := re.FindString(fr)
	re = regexp.MustCompile(`      "op": "([^"]+)",`)
	spans := re.FindAllStringSubmatch(spansField, -1)
	var ops []string
	for _, span := range spans {
		ops = append(ops, span[1])
	}
	log.Printf("spans (%d): %s", len(spans), strings.Join(ops, ", "))
	if len(spans) < 20 {
		panic(fmt.Errorf("too few spans (%d), missing database instrumentation?", len(spans)))
	}
}

func sanityCheckFakeRelayOpenTelemetry(r ResultData) {
	m := r.TestResult.RelayMetrics
	if got := m.BytesReceived; got <= 0 {
		panic(fmt.Errorf("fakerelay got %d bytes, want >0", got))
	}
	fr := m.FirstRequest
	if !strings.HasPrefix(fr, "POST /api/v2/spans") {
		panic(fmt.Errorf("fakerelay bad first request"))
	}
	i := strings.Index(fr, "\r\n\r\n")
	if i < 0 {
		panic(fmt.Errorf("invalid HTTP request: %q", fr))
	}
	body := fr[i:]
	dec := json.NewDecoder(strings.NewReader(body))
	type OpenTelemetrySpan struct {
		TraceID string `json:"traceId"`
		Kind    string `json:"kind"`
	}
	var payload []OpenTelemetrySpan
	if err := dec.Decode(&payload); err != nil {
		panic(err)
	}
	if len(payload) == 0 {
		panic("missing spans")
	}
	firstTraceID := payload[0].TraceID
	var nclient, nserver, nother int
	var kinds []string
	for _, span := range payload {
		if span.TraceID == firstTraceID {
			kinds = append(kinds, span.Kind)
			switch strings.ToLower(span.Kind) {
			case "client":
				nclient++
			case "server":
				nserver++
			default:
				nother++
			}
		}
	}
	log.Printf("spans (%d): %s", len(kinds), strings.Join(kinds, ", "))
	if nclient < 20 {
		panic(fmt.Errorf("too few CLIENT spans (%d), missing database instrumentation?", nclient))
	}
	if nserver < 1 {
		panic(fmt.Errorf("too few SERVER spans (%d), missing request handler instrumentation?", nserver))
	}
	if nother > 0 {
		log.Printf("warning: got %d spans that are neither SERVER (incoming request) nor CLIENT (outgoing database call)", nother)
	}
}
