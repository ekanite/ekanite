package status

import (
	"net"
	"time"
)

// Service provides HTTP status service.
type Service struct {
	addr string       // Bind address of the HTTP service.
	ln   net.Listener // Service listener

	start time.Time // Start up time.

	BuildInfo map[string]interface{}

	logger *log.Logger
}

// NewService returns an initialized Service object.
func NewService(addr string) *Service {
	return &Service{
		addr:   addr,
		store:  store,
		start:  time.Now(),
		logger: log.New(os.Stderr, "[status] ", log.LstdFlags),
	}
}

// Start starts the service.
func (s *Service) Start() error {
	server := http.Server{
		Handler: s,
	}

	ln, err = net.Listen("tcp", s.addr)
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
