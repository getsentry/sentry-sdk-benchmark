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
	var errors []error
	for _, rr := range r {
		for _, e := range sanityCheckOne(rr) {
			errors = append(errors, e)
			log.Print(e)
		}
	}
	if n := len(errors); n > 0 {
		panic(fmt.Errorf("%d failures", n))
	}
}

func sanityCheckOne(r ResultData) []error {
	log.Print("Sanity check: ", r.Name)
	var errors []error

	errors = append(errors, sanityCheckTargetApp(r)...)

	errors = append(errors, sanityCheckLoadGenerator(r)...)

	switch r.Name {
	case "baseline":
		errors = append(errors, sanityCheckFakeRelayBaseline(r)...)
	case "instrumented":
		errors = append(errors, sanityCheckFakeRelayInstrumented(r)...)
	case "opentelemetry":
		errors = append(errors, sanityCheckFakeRelayOpenTelemetry(r)...)
	default:
		errors = append(errors, fmt.Errorf("unexpected name: %q", r.Name))
	}

	if len(errors) == 0 {
		log.Print("Sanity check: OK")
	} else {
		log.Print("Sanity check: FAIL")
	}
	return errors
}

func sanityCheckTargetApp(r ResultData) []error {
	i := strings.Index(r.TestResult.FirstAppResponse, "\r\n\r\n")
	if i < 0 {
		return []error{fmt.Errorf("invalid HTTP response: %q", r.TestResult.FirstAppResponse)}
	}
	body := r.TestResult.FirstAppResponse[i:]
	dec := json.NewDecoder(strings.NewReader(body))
	var payload []struct {
		ID           int
		RandomNumber int
	}
	if err := dec.Decode(&payload); err != nil {
		return []error{err}
	}
	if len(payload) != 10 {
		return []error{fmt.Errorf("first app response returned %d items, want 10", len(payload))}
	}
	return nil
}

func sanityCheckLoadGenerator(r ResultData) []error {
	var errors []error
	if r.ThroughputDifferent {
		errors = append(errors, fmt.Errorf("unexpected throughput"))
	}
	if n := len(r.TestResult.Metrics.Errors); n > 0 {
		errors = append(errors, fmt.Errorf("load generator observed %d errors", n))
	}
	return errors
}

func sanityCheckFakeRelayBaseline(r ResultData) []error {
	var errors []error
	var empty RelayMetrics
	if m := r.TestResult.RelayMetrics; m != empty {
		errors = append(errors, fmt.Errorf("unexpected relay metrics: %q", m))
	}
	return errors
}

func sanityCheckFakeRelayInstrumented(r ResultData) []error {
	var errors []error
	m := r.TestResult.RelayMetrics
	if got := m.BytesReceived; got <= 0 {
		errors = append(errors, fmt.Errorf("fakerelay got %d bytes, want >0", got))
	}
	// TODO: we cannot assert on the exact number of requests
	// because r.TestResult.Requests is the number of requests sent
	// during the test, while m.Requests is the number of requests
	// received by fakerelay including the initial readiness check
	// and warmup phases.
	// In addition to that, there's a race between loadgen fetching data
	// from fakerelay and the target app making requests to fakerelay. It
	// means that m.Requests may not include all requests that the target
	// app would eventually make after the fakerelay metrics have been
	// fetched.
	if got, want := m.Requests, int(r.TestResult.Requests); got < want {
		log.Printf("warning: fakerelay got %d requests, want >=%d", got, want)
	}
	fr := m.FirstRequest
	if !strings.HasPrefix(fr, "POST /api/1/envelope") {
		errors = append(errors, fmt.Errorf("fakerelay bad first request"))
		return errors
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
		errors = append(errors, fmt.Errorf("too few spans (%d), missing database instrumentation?", len(spans)))
	}
	return errors
}

func sanityCheckFakeRelayOpenTelemetry(r ResultData) []error {
	var errors []error
	m := r.TestResult.RelayMetrics
	if got := m.BytesReceived; got <= 0 {
		errors = append(errors, fmt.Errorf("fakerelay got %d bytes, want >0", got))
	}
	fr := m.FirstRequest
	if !strings.HasPrefix(fr, "POST /api/v2/spans") {
		errors = append(errors, fmt.Errorf("fakerelay bad first request"))
		return errors
	}
	i := strings.Index(fr, "\r\n\r\n")
	if i < 0 {
		errors = append(errors, fmt.Errorf("invalid HTTP request: %q", fr))
		return errors
	}
	body := fr[i:]
	dec := json.NewDecoder(strings.NewReader(body))
	type OpenTelemetrySpan struct {
		TraceID string `json:"traceId"`
		Kind    string `json:"kind"`
	}
	var payload []OpenTelemetrySpan
	if err := dec.Decode(&payload); err != nil {
		errors = append(errors, err)
		return errors
	}
	if len(payload) == 0 {
		errors = append(errors, fmt.Errorf("missing spans"))
		return errors
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
		errors = append(errors, fmt.Errorf("too few CLIENT spans (%d), missing database instrumentation?", nclient))
	}
	if nserver < 1 {
		errors = append(errors, fmt.Errorf("too few SERVER spans (%d), missing request handler instrumentation?", nserver))
	}
	if nother > 0 {
		log.Printf("warning: got %d spans that are neither SERVER (incoming request) nor CLIENT (outgoing database call)", nother)
	}
	return errors
}
