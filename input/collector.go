package input

import (
	"bufio"
	"crypto/tls"
	"expvar"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ekanite/ekanite/input/delimiter"
)

var sequenceNumber int64
var stats = expvar.NewMap("input")

func init() {
	sequenceNumber = time.Now().UnixNano()
}

const (
	newlineTimeout = time.Duration(1000 * time.Millisecond)
	msgBufSize     = 256
)

// Collector specifies the interface all network collectors must implement.
type Collector interface {
	Start(chan<- *Event) error
	Addr() net.Addr
}

// TCPCollector represents a network collector that accepts and handler TCP connections.
type TCPCollector struct {
	iface        string
	channel      chan<- *Event
	conn         net.Conn
	fmt          string
	parser       *RFC5424Parser
	delimiter    *delimiter.Delimiter
	fDelimiter   *delimiter.FallbackDelimiter
	fallbackMode bool

	addr      net.Addr
	tlsConfig *tls.Config
}

// UDPCollector represents a network collector that accepts UDP packets.
type UDPCollector struct {
	addr   *net.UDPAddr
	fmt    string
	parser *RFC5424Parser
}

// NewCollector returns a network collector of the specified type, that will bind
// to the given inteface on Start(). If config is non-nil, a secure Collector will
// be returned. Secure Collectors require the protocol be TCP.
func NewCollector(proto, iface, format string, tlsConfig *tls.Config) (Collector, error) {
	if format != "syslog" {
		return nil, fmt.Errorf("unsupported collector format")
	}
	if strings.ToLower(proto) == "tcp" {
		return &TCPCollector{
			iface:        iface,
			fmt:          format,
			parser:       NewRFC5424Parser(),
			delimiter:    delimiter.NewDelimiter(),
			fDelimiter:   delimiter.NewFallbackDelimiter(msgBufSize),
			tlsConfig:    tlsConfig,
			fallbackMode: false,
		}, nil
	} else if strings.ToLower(proto) == "udp" {
		addr, err := net.ResolveUDPAddr("udp", iface)
		if err != nil {
			return nil, err
		}

		return &UDPCollector{addr: addr, fmt: format, parser: NewRFC5424Parser()}, nil
	}
	return nil, fmt.Errorf("unsupport collector protocol")
}

// Start instructs the TCPCollector to bind to the interface and accept connections.
func (s *TCPCollector) Start(c chan<- *Event) error {
	var ln net.Listener
	var err error
	if s.tlsConfig == nil {
		ln, err = net.Listen("tcp", s.iface)
	} else {
		ln, err = tls.Listen("tcp", s.iface, s.tlsConfig)
	}
	s.addr = ln.Addr()

	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			go s.handleConnection(conn, c)
		}
	}()
	return nil
}

// Addr returns the net.Addr that the Collector is bound to, in a race-say manner.
func (s *TCPCollector) Addr() net.Addr {
	return s.addr
}

func (s *TCPCollector) handleConnection(conn net.Conn, c chan<- *Event) {
	stats.Add("tcpConnections", 1)
	s.conn = conn
	s.channel = c
	defer func() {
		stats.Add("tcpConnections", -1)
		s.conn.Close()
	}()
	reader := bufio.NewReader(conn)
	for {
		conn.SetReadDeadline(time.Now().Add(newlineTimeout))
		b, err := reader.ReadByte()
		if err != nil {
			stats.Add("tcpConnReadError", 1)
			if !s.recover(err) {
				return
			}
		} else {
			stats.Add("tcpBytesRead", 1)
			if s.fallbackMode {
				s.useFallbackDelimiter(b)
			} else {
				s.useDelimiter(b)
			}
		}
	}
}

// Takes use of the standard delimiter
// and switches to the fallback delimiter in case
// of occuring errors.
func (s *TCPCollector) useDelimiter(b byte) {
	match, err := s.delimiter.Push(b)
	if err != nil {
		s.fallbackMode = true
		s.delimiter.Reset()
		if s.delimiter.Result != "" {
			for i := 0; i < len(s.delimiter.Result); i++ {
				s.useFallbackDelimiter(s.delimiter.Result[i])
			}
		} else {
			s.useFallbackDelimiter(b)
		}
	}
	if match {
		stats.Add("tcpEventsRx", 1)
		s.forwardLog(s.delimiter.Result)
	}
}

// Takes use of the fallback delimiter.
func (s *TCPCollector) useFallbackDelimiter(b byte) {
	log, match := s.fDelimiter.Push(b)
	if match {
		stats.Add("tcpEventsRx", 1)
		s.forwardLog(log)
	}
}

// Tries to revover from occuring network errors.
func (s *TCPCollector) recover(err error) bool {
	if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
		stats.Add("tcpConnReadTimeout", 1)
	} else if err == io.EOF {
		stats.Add("tcpConnReadEOF", 1)
	} else {
		stats.Add("tcpConnUnrecoverError", 1)
		return false
	}
	if !s.fallbackMode {
		s.delimiter.Reset()
		for i := 0; i < len(s.delimiter.Result); i++ {
			s.useFallbackDelimiter(s.delimiter.Result[i])
		}
	} else {
		log, match := s.fDelimiter.Vestige()
		if match {
			s.forwardLog(log)
		}
	}
	return true
}

// Sends the parsed log via the provided channel.
func (s *TCPCollector) forwardLog(log string) {
	s.channel <- &Event{
		Text:          log,
		Parsed:        s.parser.Parse(log),
		ReceptionTime: time.Now().UTC(),
		Sequence:      atomic.AddInt64(&sequenceNumber, 1),
		SourceIP:      s.conn.RemoteAddr().String(),
	}
}

// Start instructs the UDPCollector to start reading packets from the interface.
func (s *UDPCollector) Start(c chan<- *Event) error {
	conn, err := net.ListenUDP("udp", s.addr)
	if err != nil {
		return err
	}

	go func() {
		buf := make([]byte, msgBufSize)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			stats.Add("udpBytesRead", int64(n))
			if err != nil {
				continue
			}
			log := strings.Trim(string(buf[:n]), "\r\n")
			stats.Add("udpEventsRx", 1)
			c <- &Event{
				Text:          log,
				Parsed:        s.parser.Parse(log),
				ReceptionTime: time.Now().UTC(),
				Sequence:      atomic.AddInt64(&sequenceNumber, 1),
				SourceIP:      addr.String(),
			}
		}
	}()
	return nil
}

// Addr returns the net.Addr to which the UDP collector is bound.
func (s *UDPCollector) Addr() net.Addr {
	return s.addr
}
