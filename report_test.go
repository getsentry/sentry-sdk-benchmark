package main

import (
	"reflect"
	"testing"
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
					sdk := make(map[string]string)
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
					sdk := make(map[string]string)
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
				SdkVersion: "1.3.0",
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
