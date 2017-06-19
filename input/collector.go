package input

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"expvar"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"
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
	iface  string
	format string

	addr      net.Addr
	tlsConfig *tls.Config
}

// UDPCollector represents a network collector that accepts UDP packets.
type UDPCollector struct {
	format string
	addr   *net.UDPAddr
}

// NewCollector returns a network collector of the specified type, that will bind
// to the given inteface on Start(). If config is non-nil, a secure Collector will
// be returned. Secure Collectors require the protocol be TCP.
func NewCollector(proto, iface, format string, tlsConfig *tls.Config) (Collector, error) {
	// Verify that a parser can be instantiated. The actual parser that is used will
	// be created by the connection handler.
	_, err := NewParser(format)
	if err != nil {
		return nil, err
	}

	if strings.ToLower(proto) == "tcp" {
		return &TCPCollector{
			iface:     iface,
			format:    format,
			tlsConfig: tlsConfig,
		}, nil
	} else if strings.ToLower(proto) == "udp" {
		addr, err := net.ResolveUDPAddr("udp", iface)
		if err != nil {
			return nil, err
		}

		return &UDPCollector{addr: addr, format: format}, nil
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
	defer func() {
		stats.Add("tcpConnections", -1)
		conn.Close()
	}()

	parser, err := NewParser(s.format)
	if err != nil {
		panic(fmt.Sprintf("failed to create TCP connection parser:%s", err.Error()))
	}

	delimiter := NewSyslogDelimiter(msgBufSize)
	reader := bufio.NewReader(conn)
	var log string
	var match bool

	for {
		conn.SetReadDeadline(time.Now().Add(newlineTimeout))
		b, err := reader.ReadByte()
		if err != nil {
			stats.Add("tcpConnReadError", 1)
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				stats.Add("tcpConnReadTimeout", 1)
			} else if err == io.EOF {
				stats.Add("tcpConnReadEOF", 1)
			} else {
				stats.Add("tcpConnUnrecoverError", 1)
				return
			}

			log, match = delimiter.Vestige()
		} else {
			stats.Add("tcpBytesRead", 1)
			log, match = delimiter.Push(b)
		}

		// Log line available?
		if match {
			stats.Add("tcpEventsRx", 1)
			if parser.Parse(bytes.NewBufferString(log).Bytes()) {
				c <- &Event{
					Text:          string(parser.Raw),
					Parsed:        parser.Result,
					ReceptionTime: time.Now().UTC(),
					Sequence:      atomic.AddInt64(&sequenceNumber, 1),
					SourceIP:      conn.RemoteAddr().String(),
				}
			}
		}

		// Was the connection closed?
		if err == io.EOF {
			return
		}
	}
}

// Start instructs the UDPCollector to start reading packets from the interface.
func (s *UDPCollector) Start(c chan<- *Event) error {
	conn, err := net.ListenUDP("udp", s.addr)
	if err != nil {
		return err
	}

	parser, err := NewParser(s.format)
	if err != nil {
		panic(fmt.Sprintf("failed to create UDP parser:%s", err.Error()))
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
			if parser.Parse(bytes.NewBufferString(log).Bytes()) {
				c <- &Event{
					Text:          log,
					Parsed:        parser.Result,
					ReceptionTime: time.Now().UTC(),
					Sequence:      atomic.AddInt64(&sequenceNumber, 1),
					SourceIP:      addr.String(),
				}
			}
			stats.Add("udpEventsRx", 1)
		}
	}()
	return nil
}

// Addr returns the net.Addr to which the UDP collector is bound.
func (s *UDPCollector) Addr() net.Addr {
	return s.addr
}
