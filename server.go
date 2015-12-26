package ekanite

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

type Searcher interface {
	Search(query string) (<-chan string, error)
}

// Server serves query client connections.
type Server struct {
	iface    string
	searcher Searcher

	addr net.Addr

	Logger *log.Logger
}

// NewServer returns a new Server instance.
func NewServer(iface string, searcher Searcher) *Server {
	return &Server{
		iface:    iface,
		searcher: searcher,
		Logger:   log.New(os.Stderr, "[server] ", log.LstdFlags),
	}
}

// Start instructs the Server to bind to the interface and accept connections.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.iface)
	if err != nil {
		return err
	}
	s.addr = ln.Addr()
	http.HandleFunc("/", handler)
	go http.Serve(ln, nil)

	return nil
}

// Addr returns the address to which the Server is bound.
func (s *Server) Addr() net.Addr {
	return s.addr
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there! %s", r.URL.Path[1:])
}
