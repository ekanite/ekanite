package main

import (
	"bytes"
	"expvar"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ekanite/ekanite"
	"github.com/ekanite/ekanite/input"
)

var (
	stats = expvar.NewMap("ekanite")
)

// Program parameters
var datadir string
var tcpIface string
var udpIface string
var queryIface string
var batchSize int
var batchTimeout int
var indexMaxPending int
var gomaxprocs int
var numShards int
var retentionPeriod string
var noReport bool

// Flag set
var fs *flag.FlagSet

// Types
const (
	DefaultDataDir         = "/var/opt/ekanite"
	DefaultBatchSize       = 300
	DefaultBatchTimeout    = 1000
	DefaultIndexMaxPending = 1000
	DefaultNumShards       = 16
	DefaultRetentionPeriod = "168h"
	DefaultQueryAddr       = "localhost:9950"
	DefaultTCPServer       = "localhost:5514"
)

func main() {
	fs = flag.NewFlagSet("", flag.ExitOnError)
	var (
		datadir         = fs.String("datadir", DefaultDataDir, "Set data directory.")
		batchSize       = fs.Int("batchsize", DefaultBatchSize, "Indexing batch size.")
		batchTimeout    = fs.Int("batchtime", DefaultBatchTimeout, "Indexing batch timeout, in milliseconds.")
		indexMaxPending = fs.Int("maxpending", DefaultIndexMaxPending, "Maximum pending index events.")
		tcpIface        = fs.String("tcp", DefaultTCPServer, "Syslog server TCP bind address in the form host:port. If empty, not started.")
		udpIface        = fs.String("udp", "", "Syslog server UDP bind address in the form host:port. If not set, not started.")
		diagIface       = fs.String("diag", "", "expvar and pprof bind address in the form host:port. If not set, not started.")
		queryIface      = fs.String("query", DefaultQueryAddr, "TCP Bind address for query server in the form host:port.")
		numShards       = fs.Int("numshards", DefaultNumShards, "Set number of shards per index.")
		retentionPeriod = fs.String("retention", DefaultRetentionPeriod, "Data retention period. Minimum is 24 hours.")
		noReport        = fs.Bool("noreport", false, "Do not report anonymous data on launch.")
	)
	fs.Usage = printHelp
	fs.Parse(os.Args[1:])

	absDataDir, err := filepath.Abs(*datadir)
	if err != nil {
		log.Fatalf("failed to get absolute data path for '%s': %s", *datadir, err.Error())
	}

	// Get the retention period.
	retention, err := time.ParseDuration(*retentionPeriod)
	if err != nil {
		log.Fatalf("failed to parse retention period '%s'", *retentionPeriod)
	}

	log.SetFlags(log.LstdFlags)
	log.SetPrefix("[ekanite] ")
	log.Printf("ekanite started using %s for index storage", absDataDir)

	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Println("GOMAXPROCS set to", runtime.GOMAXPROCS(0))

	// Start the expvar handler if requested.
	if *diagIface != "" {
		sock, err := net.Listen("tcp", *diagIface)
		if err != nil {
			log.Fatalf("failed to create diag server: %s", err.Error())
		}
		go func() {
			log.Printf("diags now available at %s", *diagIface)
			http.Serve(sock, nil)
		}()
	}

	// Create and open the Engine.
	engine := ekanite.NewEngine(absDataDir)
	if engine == nil {
		log.Fatalf("failed to create indexing engine at %s", absDataDir)
	}
	engine.NumShards = *numShards
	engine.RetentionPeriod = retention

	if err := engine.Open(); err != nil {
		log.Fatalf("failed to open engine: %s", err.Error())
	}
	log.Printf("engine opened with shard number of %d, retention period of %s",
		engine.NumShards, engine.RetentionPeriod)

	// Start the query server.
	server := ekanite.NewServer(*queryIface, engine)
	if server == nil {
		log.Fatal("failed to create query server")
	}
	if err := server.Start(); err != nil {
		log.Fatalf("failed to start query server: %s", err.Error())
	}
	log.Printf("query server listening to %s", *queryIface)

	// Create and start the batcher.
	batcherTimeout := time.Duration(*batchTimeout) * time.Millisecond
	batcher := ekanite.NewBatcher(engine, *batchSize, batcherTimeout, *indexMaxPending)
	if batcher == nil {
		log.Fatal("failed to create indexing batcher")
	}

	errChan := make(chan error)
	if err := batcher.Start(errChan); err != nil {
		log.Fatalf("failed to start indexing batcher: %s", err.Error())
	}
	log.Printf("batching configured with size %d, timeout %s, max pending %d",
		*batchSize, batcherTimeout, *indexMaxPending)

	// Start draining batcher errors.
	go func() {
		for {
			select {
			case err := <-errChan:
				if err != nil {
					log.Printf("error indexing batch: %s", err.Error())
				}
			}
		}
	}()

	// Start TCP collector if requested.
	if *tcpIface != "" {
		collector := input.NewCollector("tcp", *tcpIface)
		if collector == nil {
			log.Fatalf("failed to created TCP collector bound to %s", *tcpIface)
		}
		if err := collector.Start(batcher.C()); err != nil {
			log.Fatalf("failed to start TCP collector: %s", err.Error())
		}
		log.Printf("TCP collector listening to %s", *tcpIface)
	}

	// Start UDP collector if requested.
	if *udpIface != "" {
		collector := input.NewCollector("udp", *udpIface)
		if collector == nil {
			log.Fatalf("failed to created UDP collector for to %s", *udpIface)
		}
		if err := collector.Start(batcher.C()); err != nil {
			log.Fatalf("failed to start UDP collector: %s", err.Error())
		}
		log.Printf("UDP collector listening to %s", *tcpIface)
	}

	if !*noReport {
		reportLaunch()
	}

	stats.Set("launch", time.Now().UTC())

	// Spin forever
	select {}
}

func printHelp() {
	fmt.Println("ekanite [options]")
	fs.PrintDefaults()
}

func reportLaunch() {
	json := fmt.Sprintf(`{"os": "%s", "arch": "%s", "gomaxprocs": %d, "numcpu": %d, "numshards": %d, "app": "ekanited"}`,
		runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0), runtime.NumCPU(), numShards)
	data := bytes.NewBufferString(json)
	client := http.Client{Timeout: time.Duration(5 * time.Second)}
	go client.Post("https://logs-01.loggly.com/inputs/8a0edd84-92ba-46e4-ada8-c529d0f105af/tag/reporting/",
		"application/json", data)
}
