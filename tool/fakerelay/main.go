package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"expvar"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"sync"
	"time"
)

var (
	requestCount = expvar.NewInt("requests")

	firstRequestOnce sync.Once
	firstRequest     = expvar.NewString("first_request")
	sdkInfo          SDKInfo

	bytesReceived = expvar.NewInt("bytes_received")
)

func init() {
	expvar.Publish("sdk", expvar.Func(func() interface{} {
		return sdkInfo
	}))
}

func main() {
	// Listen on all network interfaces when containerized (PID 1), or
	// otherwise only on localhost (for testing)
	addr := ":5000"
	if os.Getpid() != 1 {
		addr = "localhost" + addr
	}

	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lmsgprefix)
	log.SetPrefix("[fakerelay] ")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestCount.Add(1)

		chunked := len(r.TransferEncoding) > 0 && r.TransferEncoding[0] == "chunked"
		if chunked {
			// Remove "chunked" such that httputil.DumpRequest will
			// dump a regular body, making it easier to read JSON
			// values for debugging.
			r.TransferEncoding = r.TransferEncoding[1:]
			b, err := io.ReadAll(r.Body)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				http.Error(w, err.Error(), 500)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(b))
			r.ContentLength = int64(len(b))
		}
		if r.ContentLength < 0 {
			err := fmt.Errorf("unexpected Content-Length: %d", r.ContentLength)
			fmt.Fprintln(os.Stderr, err)
			http.Error(w, err.Error(), 500)
			return
		}
		bytesReceived.Add(r.ContentLength)

		firstRequestOnce.Do(func() {
			if r.Header.Get("Content-Encoding") == "gzip" {
				reader, err := gzip.NewReader(r.Body)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					http.Error(w, err.Error(), 500)
					return
				}
				r.Body = reader
			}
			b, err := httputil.DumpRequest(r, true)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				http.Error(w, err.Error(), 500)
				return
			}
			firstRequest.Set(string(b))
			sdkInfo = ParseSDKInfo(b)
		})

		w.Header().Add("Content-Type", "application/json")
		time.Sleep(80*time.Millisecond - time.Since(start))
		fmt.Fprint(w, `{"id":"9f95bedf1f4c4487b1b4fa8eb384b48e"}`)
	})

	log.Fatal((&http.Server{Addr: addr, BaseContext: func(l net.Listener) context.Context {
		log.Print("Serving on http://", l.Addr().String())
		return context.Background()
	}}).ListenAndServe())
}
