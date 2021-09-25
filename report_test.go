package main

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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

func TestFormatHTTP(t *testing.T) {
	tests := []struct {
		Name string
		Req  string
		Want string
	}{
		{
			Name: "Empty string",
			Req:  "",
			Want: "",
		},
		{
			Name: "No body",
			Req:  "HEAD / HTTP 1.0",
			Want: "HEAD / HTTP 1.0",
		},
		{
			Name: "Non-JSON body",
			Req: "GET / HTTP 1.1\r\n\r\n" +
				"<html></html>",
			Want: "GET / HTTP 1.1\r\n\r\n" +
				"<html></html>",
		},
		{
			Name: "Sentry envelope", // multiple line-separated JSON objects
			Req: "POST /api/1/envelope/ HTTP/1.1\r\n" +
				"Host: relay:5000\r\n" +
				"Content-Length: 1514\r\n" +
				"Content-Type: application/x-sentry-envelope\r\n" +
				"Sentry-Trace: 24077aaa50fa477c848c2b28af6fe9ce-a0bb7bbdb148ae20-\r\n" +
				"User-Agent: sentry.python/1.3.0\r\n" +
				"X-Sentry-Auth: Sentry sentry_key=sentry, sentry_version=7, sentry_client=sentry.python/1.3.0\r\n\r\n" +
				`{"event_id":"9c0f672b97e4463ebe92af551a064ceb","sent_at":"2021-09-24T23:01:24.230070Z"}` + "\n" +
				`{"type":"transaction","content_type":"application/json","length":8487}` + "\n" +
				`{"type":"transaction","transaction":"/update","contexts":{"trace":{"trace_id":"c81fbb46bbef431da70909d59954bfbb","span_id":"96114f6cf78209b8","parent_span_id":null,"op":"http.server","description":null,"status":"ok"},"runtime":{"name":"CPython","version":"3.9.1"}}}`,
			Want: "POST /api/1/envelope/ HTTP/1.1\r\n" +
				"Host: relay:5000\r\n" +
				"Content-Length: 1514\r\n" +
				"Content-Type: application/x-sentry-envelope\r\n" +
				"Sentry-Trace: 24077aaa50fa477c848c2b28af6fe9ce-a0bb7bbdb148ae20-\r\n" +
				"User-Agent: sentry.python/1.3.0\r\n" +
				"X-Sentry-Auth: Sentry sentry_key=sentry, sentry_version=7, sentry_client=sentry.python/1.3.0\r\n\r\n" +
				`{
  "event_id": "9c0f672b97e4463ebe92af551a064ceb",
  "sent_at": "2021-09-24T23:01:24.230070Z"
}
{
  "type": "transaction",
  "content_type": "application/json",
  "length": 8487
}
{
  "type": "transaction",
  "transaction": "/update",
  "contexts": {
    "trace": {
      "trace_id": "c81fbb46bbef431da70909d59954bfbb",
      "span_id": "96114f6cf78209b8",
      "parent_span_id": null,
      "op": "http.server",
      "description": null,
      "status": "ok"
    },
    "runtime": {
      "name": "CPython",
      "version": "3.9.1"
    }
  }
}
`,
		},
		{
			Name: "Zipkin spans", // single JSON array
			Req: "POST /api/v2/spans HTTP/1.1\r\n" +
				"Host: relay:5000\r\n" +
				"Content-Length: 33123\r\n" +
				"Content-Type: application/json\r\n" +
				"User-Agent: python-requests/2.26.0\r\n\r\n" +
				`[{"traceId": "743ff6051814ca3357ef9852c599fc82", "id": "739fe5bba76a3a7c", "name": "^update$", "timestamp": 1632518156885583, "duration": 2006248, "localEndpoint": {"serviceName": "unknown_service"}, "kind": "SERVER", "tags": {"http.method": "GET"}, "debug": true}]`,
			Want: "POST /api/v2/spans HTTP/1.1\r\n" +
				"Host: relay:5000\r\n" +
				"Content-Length: 33123\r\n" +
				"Content-Type: application/json\r\n" +
				"User-Agent: python-requests/2.26.0\r\n\r\n" +
				`[
  {
    "traceId": "743ff6051814ca3357ef9852c599fc82",
    "id": "739fe5bba76a3a7c",
    "name": "^update$",
    "timestamp": 1632518156885583,
    "duration": 2006248,
    "localEndpoint": {
      "serviceName": "unknown_service"
    },
    "kind": "SERVER",
    "tags": {
      "http.method": "GET"
    },
    "debug": true
  }
]
`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			got := formatHTTP(tt.Req)
			if diff := cmp.Diff(tt.Want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
