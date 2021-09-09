package main

import (
	"strings"
	"testing"
	"time"
)

func TestWriteStat(t *testing.T) {
	var b strings.Builder

	writeStat(&b, BenchstatPrefix, "Latencies", "1", time.Duration(11703760800).Milliseconds())

	got := b.String()
	want := "BenchmarkLatencies 1 11703 ms\n"

	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
