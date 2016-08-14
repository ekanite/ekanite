package ekanite

import (
	"bufio"
	"log"
	"net"
	"os"
	"strings"
)

// Searcher is the interface any object that perform searches should implement.
type Searcher interface {
	Search(query string) (<-chan string, error)
}

// Server serves query client connections.
type Server struct {
	iface    string
	Searcher Searcher

	addr net.Addr

	Logger *log.Logger
}

// NewServer returns a new Server instance.
func NewServer(iface string, searcher Searcher) *Server {
	return &Server{
		iface:    iface,
		Searcher: searcher,
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

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			go s.handleConnection(conn)
		}
	}()
	return nil
}

// Addr returns the address to which the Server is bound.
func (s *Server) Addr() net.Addr {
	return s.addr
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		s.Logger.Printf("connection from %s closed", conn.RemoteAddr())
	}()
	s.Logger.Printf("new connection from %s", conn.RemoteAddr())

	reader := bufio.NewReader(conn)
	for {
		b, err := reader.ReadString('\n')
		if err != nil {
			conn.Close()
			return
		}

		query := strings.Trim(b, "\r\n")
		if query == "" {
			continue
		}

		s.Logger.Printf("executing query '%s'", query)
		c, err := s.Searcher.Search(query)
		if err != nil {
			conn.Write([]byte(err.Error()))
		} else {
			for s := range c {
				conn.Write([]byte(s + "\n"))
			}
		}
		// Send two newlines to indicate end-of-results.
		conn.Write([]byte("\n\n"))
	}
}
