package status

import (
	"encoding/json"
	"expvar"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"
	"sync"
	"time"
)

// Provider is the interface status providers should implement.
type Provider interface {
	Status() (map[string]interface{}, error)
}

// Service provides HTTP status service.
type Service struct {
	addr string       // Bind address of the HTTP service.
	ln   net.Listener // Service listener

	start     time.Time           // Start up time.
	providers map[string]Provider // Registered providers
	mu        sync.Mutex

	BuildInfo map[string]interface{}

	logger *log.Logger
}

// NewService returns an initialized Service object.
func NewService(addr string) *Service {
	return &Service{
		addr:      addr,
		start:     time.Now(),
		providers: make(map[string]Provider),
		logger:    log.New(os.Stderr, "[status] ", log.LstdFlags),
	}
}

// Start starts the service.
func (s *Service) Start() error {
	server := http.Server{
		Handler: s,
	}

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.ln = ln

	go func() {
		err := server.Serve(s.ln)
		if err != nil {
			s.logger.Println("HTTP service Serve() returned:", err.Error())
		}
	}()
	s.logger.Println("service listening on", s.addr)

	return nil
}

// Close closes the service.
func (s *Service) Close() {
	s.ln.Close()
	return
}

// Addr returns the address on which the Service is listening
func (s *Service) Addr() net.Addr {
	return s.ln.Addr()
}

// Register registers the given provider with the given key. Calls to register
// providers on uninitialized services will be ignored.
func (s *Service) Register(key string, provider Provider) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[key] = provider
	s.logger.Println("status provider registered for", key)
}

// ServeHTTP allows Service to serve HTTP requests.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Add version header to every response, if available.
	if v, ok := s.BuildInfo["version"].(string); ok {
		w.Header().Add("X-EKANITE-VERSION", v)
	} else {
		w.Header().Add("X-EKANITE-VERSION", "unknown")
	}

	switch {
	case strings.HasPrefix(r.URL.Path, "/status"):
		s.handleStatus(w, r)
	case r.URL.Path == "/debug/vars":
		serveExpvar(w, r)
	case strings.HasPrefix(r.URL.Path, "/debug/pprof"):
		servePprof(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// handleStatus returns status on the system.
func (s *Service) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := make(map[string]interface{})
	for k, p := range s.providers {
		st, err := p.Status()
		if err != nil {
			s.logger.Printf("failed to retrieve status for %s: %s", k, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		status[k] = st
	}

	pretty, _ := isPretty(r)
	var b []byte
	var err error
	if pretty {
		b, err = json.MarshalIndent(status, "", "    ")
	} else {
		b, err = json.Marshal(status)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		_, err = w.Write([]byte(b))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// serveExpvar serves registered expvar information over HTTP.
func serveExpvar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\n")
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}

// servePprof serves pprof information over HTTP.
func servePprof(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/debug/pprof/cmdline":
		pprof.Cmdline(w, r)
	case "/debug/pprof/profile":
		pprof.Profile(w, r)
	case "/debug/pprof/symbol":
		pprof.Symbol(w, r)
	default:
		pprof.Index(w, r)
	}
}

// queryParam returns whether the given query param is set to true.
func queryParam(req *http.Request, param string) (bool, error) {
	err := req.ParseForm()
	if err != nil {
		return false, err
	}
	if _, ok := req.Form[param]; ok {
		return true, nil
	}
	return false, nil
}

// isPretty returns whether the HTTP response body should be pretty-printed.
func isPretty(req *http.Request) (bool, error) {
	return queryParam(req, "pretty")
}
