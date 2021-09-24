package main

import (
	"reflect"
	"testing"
	"time"
)

func Test_formatSDKName(t *testing.T) {
	tests := []struct {
		n    string
		want string
	}{
		{
			n:    "sentry.python",
			want: "sentry-python",
		},
		{
			n:    "sentry.javascript.node",
			want: "sentry-javascript",
		},
	}
	for _, tt := range tests {
		t.Run(tt.n, func(t *testing.T) {
			if got := formatSDKName(tt.n); got != tt.want {
				t.Errorf("formatSDKName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getAppDetails(t *testing.T) {
	type args struct {
		path         string
		relayMetrics map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want AppDetails
	}{
		{
			name: "Python Django",
			args: args{
				path: "result/python/django/20210923-152931-snbclwa/baseline",
				relayMetrics: func() map[string]interface{} {
					sdk := make(map[string]interface{})
					sdk["name"] = "sentry.python"
					sdk["version"] = "1.3.0"

					relayMetrics := make(map[string]interface{})
					relayMetrics["sdk"] = sdk
					return relayMetrics
				}(),
			},
			want: AppDetails{
				Language:   "python",
				Framework:  "django",
				SdkName:    "sentry-python",
				SdkVersion: "1.3.0",
			},
		},
		{
			name: "JavaScript Express",
			args: args{
				path: "result/javascript/express/20210923-145159-yaubxsi/baseline",
				relayMetrics: func() map[string]interface{} {
					sdk := make(map[string]interface{})
					sdk["name"] = "sentry.javascript"
					sdk["version"] = "6.11.0"

					relayMetrics := make(map[string]interface{})
					relayMetrics["sdk"] = sdk
					return relayMetrics
				}(),
			},
			want: AppDetails{
				Language:   "javascript",
				Framework:  "express",
				SdkName:    "sentry-javascript",
				SdkVersion: "6.11.0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAppDetails(tt.args.path, tt.args.relayMetrics); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getConfiguration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_percentDiff(t *testing.T) {
	type args struct {
		start time.Duration
		final time.Duration
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "example percentage",
			args: args{
				start: 31201976,
				final: 35903631,
			},
			want: 15.07,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := percentDiff(tt.args.start, tt.args.final); got != tt.want {
				t.Errorf("percentDiff() = %v, want %v", got, tt.want)
			}
		})
	}
}
