package main

import (
	"testing"
)

func TestParseSDKInfo(t *testing.T) {
	tests := []struct {
		in   string
		want SDKInfo
	}{
		{
			in: "sentry_client=sentry.python/1.3.0\r\n\r\n{\"event_id\":\"c822b648d94e47a2b9cd7d67a9984eee\",\"sent_at\":...",
			want: SDKInfo{
				Name:    "sentry.python",
				Version: "1.3.0",
			},
		},
		{
			in: "sentry_client=sentry-curl/1.0",
			want: SDKInfo{
				Name:    "sentry.curl",
				Version: "1.0",
			},
		},
		{
			in: "sentry_client=sentry.javascript.node/6.11.0,",
			want: SDKInfo{
				Name:    "sentry.javascript.node",
				Version: "6.11.0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.want.Name, func(t *testing.T) {
			got := ParseSDKInfo([]byte(tt.in))
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
