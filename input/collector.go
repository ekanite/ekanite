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
	"sync"
	"sync/atomic"
	"time"

	"github.com/ekanite/ekanite/input/delimiter"
	"github.com/ekanite/ekanite/input/parser"
)

var (
	sequenceNumber int64
	stats                      = expvar.NewMap("input")
	mutex          *sync.Mutex = &sync.Mutex{}
)

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
	iface          string
	connRemoteAddr string
	channel        chan<- *Event
	parser         *parser.Parser
	delimiter      *delimiter.Delimiter
	fDelimiter     *delimiter.FallbackDelimiter
	fallbackMode   bool

	addr      net.Addr
	tlsConfig *tls.Config
}

// UDPCollector represents a network collector that accepts UDP packets.
type UDPCollector struct {
	addr   *net.UDPAddr
	parser *parser.Parser
}

// NewCollector returns a network collector of the specified type, that will bind
// to the given inteface on Start(). If config is non-nil, a secure Collector will
// be returned. Secure Collectors require the protocol be TCP.
func NewCollector(proto, iface, format string, tlsConfig *tls.Config) (Collector, error) {
	if !parser.IsFmt(format) {
		return nil, fmt.Errorf("unsupported collector format")
	}
	if strings.ToLower(proto) == "tcp" {
		return &TCPCollector{
			iface:        iface,
			parser:       parser.NewParser(format),
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

		return &UDPCollector{addr: addr, parser: parser.NewParser(format)}, nil
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
	mutex.Lock()
	s.connRemoteAddr = conn.RemoteAddr().String()
	s.channel = c
	mutex.Unlock()
	defer func() {
		stats.Add("tcpConnections", -1)
		conn.Close()
	}()
	reader := bufio.NewReader(conn)
	for {
		conn.SetReadDeadline(time.Now().Add(newlineTimeout))
		b, err := reader.ReadByte()
		if err != nil {
			stats.Add("tcpConnReadError", 1)
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				stats.Add("tcpConnReadTimeout", 1)
				s.recover()
			} else if err == io.EOF {
				stats.Add("tcpConnReadEOF", 1)
				s.recover()
			} else {
				stats.Add("tcpConnUnrecoverError", 1)
				return
			}
		} else {
			stats.Add("tcpBytesRead", 1)
			if s.parser.Fmt == "rfc5424" && s.fallbackMode {
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
		if s.parser.Fmt == "rfc5424" {
			s.fallbackMode = true
			s.delimiter.Reset()
			if s.delimiter.Result != "" {
				for i := 0; i < len(s.delimiter.Result); i++ {
					s.useFallbackDelimiter(s.delimiter.Result[i])
				}
			} else {
				s.useFallbackDelimiter(b)
			}
			return
		}
		stats.Add("tcpDelimiterBroken", 1)
	}
	if match {
		s.forwardLog(s.delimiter.Result)
	}
}

// Takes use of the fallback delimiter.
func (s *TCPCollector) useFallbackDelimiter(b byte) {
	log, match := s.fDelimiter.Push(b)
	if match {
		s.forwardLog(log)
	}
}

// Tries to revover from occuring network errors.
func (s *TCPCollector) recover() bool {
	if !s.fallbackMode {
		mutex.Lock()
		s.delimiter.Reset()
		for i := 0; i < len(s.delimiter.Result); i++ {
			s.useFallbackDelimiter(s.delimiter.Result[i])
		}
		mutex.Unlock()
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
	stats.Add("tcpEventsRx", 1)
	if s.parser.Parse(bytes.NewBufferString(log).Bytes()) {
		mutex.Lock()
		s.channel <- &Event{
			Text:          string(s.parser.Raw),
			Parsed:        s.parser.Result,
			ReceptionTime: time.Now().UTC(),
			Sequence:      atomic.AddInt64(&sequenceNumber, 1),
			SourceIP:      s.connRemoteAddr,
		}
		mutex.Unlock()
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
			if s.parser.Parse(bytes.NewBufferString(log).Bytes()) {
				c <- &Event{
					Text:          log,
					Parsed:        s.parser.Result,
					ReceptionTime: time.Now().UTC(),
					Sequence:      atomic.AddInt64(&sequenceNumber, 1),
					SourceIP:      addr.String(),
				}
			}
		}
	}()
	return nil
}

// Addr returns the net.Addr to which the UDP collector is bound.
func (s *UDPCollector) Addr() net.Addr {
	return s.addr
}
