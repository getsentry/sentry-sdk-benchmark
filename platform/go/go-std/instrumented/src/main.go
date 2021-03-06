package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"

	"go-std/src/instrumented/handlers"
	"go-std/src/instrumented/storage"
	"go-std/src/instrumented/templates"
)

func initSyncPools() {
	handlers.InitMessagePool()
	storage.InitWorldPool()
	storage.InitWorldsPool()
	templates.InitFortunesPool()
}

func main() {
	err := sentry.Init(sentry.ClientOptions{
		TracesSampleRate: 1.0,
	})
	if err != nil {
		panic(err)
	}
	defer sentry.Flush(2 * time.Second)

	initSyncPools()

	// init flags
	bindHost := flag.String("bind", ":8080", "set bind host")
	prefork := flag.Bool("prefork", false, "use prefork")
	easyjson := flag.Bool("easyjson", false, "use easyjson")
	child := flag.Bool("child", false, "is child proc")
	dbDriver := flag.String("db", "none", "db connection driver [values: pq || pgx || mysql || mgo || none]")
	dbConnectionString := flag.String("db_connection_string",
		"host=tfb-database user=benchmarkdbuser password=benchmarkdbpass dbname=hello_world sslmode=disable",
		"db connection string")
	flag.Parse()

	// check for prefork
	var listener net.Listener
	if *prefork {
		listener = doPrefork(*child, *bindHost)
	} else {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	// init database with appropriate driver
	db, err := storage.InitDB(*dbDriver, *dbConnectionString, runtime.NumCPU()*4)
	if err != nil {
		log.Fatal(err)
	}
	if *child {
		db, err = storage.InitDB(*dbDriver, *dbConnectionString, runtime.NumCPU())
		if err != nil {
			log.Fatal(err)
		}
	}

	// init handlers
	http.HandleFunc("/plaintext", handlers.PlaintextHandler)
	if *easyjson {
		http.HandleFunc("/json", handlers.JSONHandlerEasyJSON)
	} else {
		http.HandleFunc("/json", handlers.JSONHandler)
	}
	if db != nil {
		defer db.Close()
		// http.HandleFunc("/fortune", handlers.FortuneHandler(db))
		http.HandleFunc("/fortunes", handlers.FortuneHandlerPool(db))
		// http.HandleFunc("/fortune-quick", handlers.FortuneQuickHandler(db))
		http.HandleFunc("/fortune-quick", handlers.FortuneQuickHandlerPool(db))
		if *easyjson {
			http.HandleFunc("/db", handlers.DBHandlerEasyJSON(db))
			http.HandleFunc("/queries", handlers.QueriesHandlerEasyJSON(db))
			http.HandleFunc("/update", handlers.UpdateHandlerEasyJSON(db))
		} else {
			http.HandleFunc("/db", handlers.DBHandler(db))
			http.HandleFunc("/queries", handlers.QueriesHandler(db))
			http.HandleFunc("/update", handlers.UpdateHandler(db))
		}
	}

	handler := sentryhttp.New(sentryhttp.Options{}).Handle(http.DefaultServeMux)

	// start server
	if *prefork {
		http.Serve(listener, handler)
	} else {
		http.ListenAndServe(*bindHost, handler)
	}
}
