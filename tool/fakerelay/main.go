package main

import (
	"compress/gzip"
	"expvar"
	"fmt"
	"log"
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

	log.Print("Fake Relay")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestCount.Add(1)
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
		os.Stdout.Write(b)
		firstRequestOnce.Do(func() {
			firstRequest.Set(string(b))
			sdkInfo = ParseSDKInfo(b)
		})
		w.Header().Add("Content-Type", "application/json")
		time.Sleep(80*time.Millisecond - time.Since(start))
		fmt.Fprint(w, `{"id":"9f95bedf1f4c4487b1b4fa8eb384b48e"}`)
	})
	log.Fatal(http.ListenAndServe(addr, nil))
}
