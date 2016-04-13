package main

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"expvar"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"syscall"
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
var caPemPath string
var caKeyPath string
var queryIface string
var batchSize int
var batchTimeout int
var indexMaxPending int
var gomaxprocs int
var numShards int
var retentionPeriod string
var cpuProfile string
var memProfile string

// Flag set
var fs *flag.FlagSet

// Types
const (
	DefaultDataDir         = "/var/opt/ekanite"
	DefaultBatchSize       = 300
	DefaultBatchTimeout    = 1000
	DefaultIndexMaxPending = 1000
	DefaultNumShards       = 4
	DefaultRetentionPeriod = "168h"
	DefaultQueryAddr       = "localhost:9950"
	DefaultHTTPQueryAddr   = "localhost:8080"
	DefaultDiagsIface      = "localhost:9951"
	DefaultTCPServer       = "localhost:5514"
	DefaultInputFormat     = "syslog"
)

func main() {
	fs = flag.NewFlagSet("", flag.ExitOnError)
	var (
		datadir         = fs.String("datadir", DefaultDataDir, "Set data directory")
		batchSize       = fs.Int("batchsize", DefaultBatchSize, "Indexing batch size")
		batchTimeout    = fs.Int("batchtime", DefaultBatchTimeout, "Indexing batch timeout, in milliseconds")
		indexMaxPending = fs.Int("maxpending", DefaultIndexMaxPending, "Maximum pending index events")
		tcpIface        = fs.String("tcp", DefaultTCPServer, "Syslog server TCP bind address in the form host:port. To disable set to empty string")
		udpIface        = fs.String("udp", "", "Syslog server UDP bind address in the form host:port. If not set, not started")
		diagIface       = fs.String("diag", DefaultDiagsIface, "expvar and pprof bind address in the form host:port. If not set, not started")
		caPemPath       = fs.String("tlspem", "", "path to CA PEM file for TLS-enabled TCP server. If not set, TLS not activated")
		caKeyPath       = fs.String("tlskey", "", "path to CA key file for TLS-enabled TCP server. If not set, TLS not activated")
		queryIface      = fs.String("query", DefaultQueryAddr, "TCP Bind address for query server in the form host:port. To disable set to empty string")
		queryIfaceHttp  = fs.String("queryhttp", DefaultHTTPQueryAddr, "TCP Bind address for http query server in the form host:port. To disable set to empty string")
		numShards       = fs.Int("numshards", DefaultNumShards, "Set number of shards per index")
		retentionPeriod = fs.String("retention", DefaultRetentionPeriod, "Data retention period. Minimum is 24 hours")
		cpuProfile      = fs.String("cpuprof", "", "Where to write CPU profiling data. Not written if not set")
		memProfile      = fs.String("memprof", "", "Where to write memory profiling data. Not written if not set")
		inputFormat     = fs.String("input", DefaultInputFormat, "Message format of input.")
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

	// Start the simple query server if requested.
	if *queryIface != "" {
		server := ekanite.NewServer(*queryIface, engine)
		if server == nil {
			log.Fatal("failed to create query server")
		}
		if err := server.Start(); err != nil {
			log.Fatalf("failed to start query server: %s", err.Error())
		}
		log.Printf("query server listening to %s", *queryIface)
	}

	// Start the http query server if requested.
	if *queryIfaceHttp != "" {
		server := ekanite.NewHttpServer(*queryIfaceHttp, engine)
		if server == nil {
			log.Fatal("failed to create HTTP query server")
		}
		if err := server.Start(); err != nil {
			log.Fatalf("failed to start HTTP query server: %s", err.Error())
		}
		log.Printf("HTTP query server listening to %s", *queryIfaceHttp)
	}

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
		var tlsConfig *tls.Config
		if *caPemPath != "" && *caKeyPath != "" {
			tlsConfig, err = newTLSConfig(*caPemPath, *caKeyPath)
			if err != nil {
				log.Fatalf("failed to configure TLS: %s", err.Error())
			}
			log.Printf("TLS successfully configured")
		}

		collector := input.NewCollector("tcp", *tcpIface, tlsConfig, *inputFormat)
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
		collector := input.NewCollector("udp", *udpIface, nil, *inputFormat)
		if collector == nil {
			log.Fatalf("failed to created UDP collector for to %s", *udpIface)
		}
		if err := collector.Start(batcher.C()); err != nil {
			log.Fatalf("failed to start UDP collector: %s", err.Error())
		}
		log.Printf("UDP collector listening to %s", *udpIface)
	}

	// Start profiling.
	startProfile(*cpuProfile, *memProfile)

	stats.Set("launch", time.Now().UTC())

	// Set up signal handling.
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// Block until one of the signals above is received
	select {
	case <-signalCh:
		log.Println("signal received, shutting down...")
	}

	stopProfile()
}

func newTLSConfig(caPemPath, caKeyPath string) (*tls.Config, error) {
	var config *tls.Config

	caPem, err := ioutil.ReadFile(caPemPath)
	if err != nil {
		return nil, err
	}
	ca, err := x509.ParseCertificate(caPem)
	if err != nil {
		return nil, err
	}

	caKey, err := ioutil.ReadFile(caKeyPath)
	if err != nil {
		return nil, err
	}
	key, err := x509.ParsePKCS1PrivateKey(caKey)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca)

	cert := tls.Certificate{
		Certificate: [][]byte{caPem},
		PrivateKey:  key,
	}

	config = &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
	}

	config.Rand = rand.Reader

	return config, nil
}

// prof stores the file locations of active profiles.
var prof struct {
	cpu *os.File
	mem *os.File
}

// StartProfile initializes the cpu and memory profile, if specified.
func startProfile(cpuprofile, memprofile string) {
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatalf("cpuprofile: %v", err)
		}
		log.Printf("writing CPU profile to: %s\n", cpuprofile)
		prof.cpu = f
		pprof.StartCPUProfile(prof.cpu)
	}

	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			log.Fatalf("memprofile: %v", err)
		}
		log.Printf("writing memory profile to: %s\n", memprofile)
		prof.mem = f
		runtime.MemProfileRate = 4096
	}

}

// StopProfile closes the cpu and memory profiles if they are running.
func stopProfile() {
	if prof.cpu != nil {
		pprof.StopCPUProfile()
		prof.cpu.Close()
		log.Println("CPU profile stopped")
	}
	if prof.mem != nil {
		pprof.Lookup("heap").WriteTo(prof.mem, 0)
		prof.mem.Close()
		log.Println("memory profile stopped")
	}
}

func printHelp() {
	fmt.Println("ekanited [options]")
	fs.PrintDefaults()
}
