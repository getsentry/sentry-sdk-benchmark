package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestBenchmarkConfigFromPath(t *testing.T) {
	djangoAll := BenchmarkConfig{
		// ID: ...,
		// StartTime: ...,
		Platform: "testdata/platform/python/django",
		PlatformConfig: PlatformConfig{
			Target: struct{ Path string }{
				Path: "/update?queries=10",
			},
			RPS: 10,
		},
		Runs: []RunConfig{
			{
				Name:       "baseline",
				NeedsRelay: false,
			},
			{
				Name:       "instrumented",
				NeedsRelay: true,
			},
		},
	}
	djangoInstrumented := BenchmarkConfig{
		// ID: ...,
		// StartTime: ...,
		Platform: "testdata/platform/python/django",
		PlatformConfig: PlatformConfig{
			Target: struct{ Path string }{
				Path: "/update?queries=10",
			},
			RPS: 10,
		},
		Runs: []RunConfig{
			{
				Name:       "instrumented",
				NeedsRelay: true,
			},
		},
	}
	tests := []struct {
		Path string
		Want BenchmarkConfig
	}{
		{"testdata/platform/python/django", djangoAll},
		{"testdata/platform/python/django/instrumented", djangoInstrumented},
	}
	opts := cmpopts.IgnoreFields(BenchmarkConfig{}, "ID", "StartTime")
	for _, tt := range tests {
		tt := tt
		name := strings.TrimPrefix(tt.Path, "testdata/platform/")
		t.Run(name, func(t *testing.T) {
			want := tt.Want
			got := BenchmarkConfigFromPath(filepath.FromSlash(tt.Path))
			if diff := cmp.Diff(want, got, opts); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
